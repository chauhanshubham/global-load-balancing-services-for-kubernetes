/*
 * Copyright 2019-2020 VMware, Inc.
 * All Rights Reserved.
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*   http://www.apache.org/licenses/LICENSE-2.0
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
*/

package ingestion

import (
	"errors"

	filter "github.com/vmware/global-load-balancing-services-for-kubernetes/gslb/gdp_filter"
	"github.com/vmware/global-load-balancing-services-for-kubernetes/gslb/gslbutils"
	"github.com/vmware/global-load-balancing-services-for-kubernetes/gslb/k8sobjects"
	"github.com/vmware/global-load-balancing-services-for-kubernetes/gslb/nodes"
	gslbalphav1 "github.com/vmware/global-load-balancing-services-for-kubernetes/internal/apis/amko/v1alpha1"

	avicache "github.com/vmware/global-load-balancing-services-for-kubernetes/gslb/cache"

	"github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func fetchAndApplyAllIngresses(c *GSLBMemberController, nsList *corev1.NamespaceList) {
	var ingList []*v1beta1.Ingress

	acceptedIngStore := gslbutils.GetAcceptedIngressStore()
	rejectedIngStore := gslbutils.GetRejectedIngressStore()

	switch c.informers.IngressVersion {
	case utils.CoreV1IngressInformer:
		for _, namespace := range nsList.Items {
			objList, err := c.informers.ClientSet.NetworkingV1beta1().Ingresses(namespace.Name).List(metav1.ListOptions{})
			if err != nil {
				gslbutils.Errf("process: fullsync, namespace: %s, msg: error in fetching the ingress list, %s",
					namespace.Name, err.Error())
				continue
			}
			for _, obj := range objList.Items {
				ingObj, ok := utils.ToNetworkingIngress(&obj)
				if !ok {
					gslbutils.Errf("process: fullsync, namespace: %s, msg: unable to convert obj to ingress")
					continue
				}
				ingList = append(ingList, ingObj.DeepCopy())
			}
		}
	case utils.ExtV1IngressInformer:
		for _, namespace := range nsList.Items {
			objList, err := c.informers.ClientSet.ExtensionsV1beta1().Ingresses(namespace.Name).List(metav1.ListOptions{})
			if err != nil {
				gslbutils.Errf("process: fullsync, namespace: %s, msg: error in fetching the ingress list, %s",
					namespace.Name, err.Error())
				continue
			}
			for _, obj := range objList.Items {
				ingObj, ok := utils.ToNetworkingIngress(&obj)
				if !ok {
					gslbutils.Errf("process: fullsync, namespace: %s, msg: error in fetching the ingress list, %s",
						namespace.Name, err.Error())
					continue
				}
				ingList = append(ingList, ingObj)
			}
		}
	}
	for _, ing := range ingList {
		ihms := k8sobjects.GetIngressHostMeta(ing, c.GetName())
		filterAndAddIngressMeta(ihms, c, acceptedIngStore, rejectedIngStore, 0, true)
	}
}

func fetchAndApplyAllServices(c *GSLBMemberController, nsList *corev1.NamespaceList) {
	acceptedLBSvcStore := gslbutils.GetAcceptedLBSvcStore()
	rejectedLBSvcStore := gslbutils.GetRejectedLBSvcStore()

	for _, namespace := range nsList.Items {
		svcList, err := c.informers.ClientSet.CoreV1().Services(namespace.Name).List(metav1.ListOptions{})
		if err != nil {
			gslbutils.Errf("process: fullsync, namespace: %s, msg: error in fetching the service list, %s",
				namespace.Name, err.Error())
			continue
		}
		for _, svc := range svcList.Items {
			if !isSvcTypeLB(&svc) {
				continue
			}
			svcMeta, ok := k8sobjects.GetSvcMeta(&svc, c.GetName())
			if !ok {
				gslbutils.Logf("cluster: %s, namespace: %s, svc: %s, msg: couldn't get meta object for service",
					c.GetName(), namespace.Name, svc.Name)
				continue
			}
			if !filter.ApplyFilter(svcMeta, c.GetName()) {
				AddOrUpdateLBSvcStore(rejectedLBSvcStore, &svc, c.GetName())
				gslbutils.Logf("cluster: %s, ns: %s, svc: %s, msg: %s", c.GetName(), namespace.Name,
					svc.Name, "rejected ADD svc key because it couldn't pass through the filter")
				continue
			}
			AddOrUpdateLBSvcStore(acceptedLBSvcStore, &svc, c.GetName())
		}
	}
}

func fetchAndApplyAllRoutes(c *GSLBMemberController, nsList *corev1.NamespaceList) {
	acceptedRouteStore := gslbutils.GetAcceptedRouteStore()
	rejectedRotueStore := gslbutils.GetRejectedRouteStore()

	for _, namespace := range nsList.Items {
		routeList, err := c.informers.OshiftClient.RouteV1().Routes(namespace.Name).List(metav1.ListOptions{})
		if err != nil {
			gslbutils.Errf("process: fullsync, namespace: %s, msg: error in fetching the  list, %s",
				namespace.Name, err.Error())
			continue
		}
		for _, route := range routeList.Items {
			routeMeta := k8sobjects.GetRouteMeta(&route, c.name)
			if routeMeta.IPAddr == "" || routeMeta.Hostname == "" {
				gslbutils.Debugf("cluster: %s, ns: %s, route: %s, msg: %s", c.name, routeMeta.Namespace,
					routeMeta.Name, "rejected ADD route because IP address/hostname not found in status field")
				continue
			}
			if !filter.ApplyFilter(routeMeta, c.name) {
				AddOrUpdateRouteStore(rejectedRotueStore, &route, c.name)
				gslbutils.Logf("cluster: %s, ns: %s, route: %s, msg: %s, routeObj: %v", c.name, routeMeta.Namespace,
					routeMeta.Name, "rejected ADD route key because it couldn't pass through the filter", routeMeta)
				continue
			}
			AddOrUpdateRouteStore(acceptedRouteStore, &route, c.name)
		}
	}
}

func checkGDPsAndInitialize() error {
	gdpList, err := gslbutils.GlobalGslbClient.AmkoV1alpha1().GlobalDeploymentPolicies(gslbutils.AVISystem).List(metav1.ListOptions{})
	if err != nil {
		return nil
	}

	// if no GDP objects, then simply return
	if len(gdpList.Items) == 0 {
		return nil
	}

	// check if any of these GDP objects have "success" in their fields
	var successGDP *gslbalphav1.GlobalDeploymentPolicy

	for _, gdp := range gdpList.Items {
		if gdp.Status.ErrorStatus == GDPSuccess {
			if successGDP == nil {
				successGDP = &gdp
			} else {
				// there are more than two accepted GDPs, which pertains to an undefined state
				gslbutils.Errf("ns: %s, msg: more than one GDP objects which were accepted, undefined state, can't do a full sync",
					gslbutils.AVISystem)
				return errors.New("more than one GDP objects in accepted state")
			}
		}
	}

	if successGDP != nil {
		AddGDPObj(successGDP, nil, 0)
	}

	// no success GDPs, check if only one exists
	if len(gdpList.Items) > 1 {
		return errors.New("more than one GDP objects")
	}

	AddGDPObj(&gdpList.Items[0], nil, 0)
	return nil
}

func bootupSync(ctrlList []*GSLBMemberController, gsCache *avicache.AviCache) {
	gslbutils.Logf("Starting boot up sync, will sync all ingresses, routes and services from all member clusters")

	// add a GDP object
	err := checkGDPsAndInitialize()
	if err != nil {
		// Undefined state, panic
		panic(err.Error())
	}

	gf := gslbutils.GetGlobalFilter()

	acceptedNSStore := gslbutils.GetAcceptedNSStore()
	rejectedNSStore := gslbutils.GetRejectedNSStore()

	for _, c := range ctrlList {
		gslbutils.Logf("syncing for cluster %s", c.GetName())
		if !gf.IsClusterAllowed(c.name) {
			gslbutils.Logf("cluster %s is not allowed via GDP", c.name)
			continue
		}
		// get all namespaces
		selectedNamespaces, err := c.informers.ClientSet.CoreV1().Namespaces().List(metav1.ListOptions{})
		if err != nil {
			gslbutils.Errf("cluster: %s, error in fetching namespaces, %s", c.name, err.Error())
			return
		}

		if len(selectedNamespaces.Items) == 0 {
			gslbutils.Errf("namespaces list is empty, can't do a full sync, returning")
			return
		}

		for _, ns := range selectedNamespaces.Items {
			_, err := gf.GetNSFilterLabel()
			if err == nil {
				nsMeta := k8sobjects.GetNSMeta(&ns, c.GetName())
				if !filter.ApplyFilter(nsMeta, c.GetName()) {
					AddOrUpdateNSStore(rejectedNSStore, &ns, c.GetName())
					gslbutils.Logf("cluster: %s, ns: %s, msg: %s\n", c.GetName(), nsMeta.Name,
						"ns didn't pass through the filter, adding to rejected list")
					continue
				}
				AddOrUpdateNSStore(acceptedNSStore, &ns, c.GetName())
			} else {
				gslbutils.Debugf("no namespace filter present, will sync the applications now")
			}
		}
		if c.informers.IngressInformer != nil {
			fetchAndApplyAllIngresses(c, selectedNamespaces)
		}

		if c.informers.ServiceInformer != nil {
			fetchAndApplyAllServices(c, selectedNamespaces)
		}
		if c.informers.RouteInformer != nil {
			fetchAndApplyAllRoutes(c, selectedNamespaces)
		}
	}

	// Generate models
	GenerateModels(gsCache)
	gslbutils.Logf("boot up sync completed")
}

func GenerateModels(gsCache *avicache.AviCache) {
	gslbutils.Logf("will generate GS graphs from all accepted lists")
	acceptedIngStore := gslbutils.GetAcceptedIngressStore()
	acceptedLBSvcStore := gslbutils.GetAcceptedLBSvcStore()
	acceptedRouteStore := gslbutils.GetAcceptedRouteStore()

	ingList := acceptedIngStore.GetAllClusterNSObjects()
	for _, ingName := range ingList {
		nodes.DequeueIngestion(gslbutils.MultiClusterKeyWithObjName(gslbutils.ObjectAdd,
			gslbutils.IngressType, ingName))
	}

	svcList := acceptedLBSvcStore.GetAllClusterNSObjects()
	for _, svcName := range svcList {
		nodes.DequeueIngestion(gslbutils.MultiClusterKeyWithObjName(gslbutils.ObjectAdd,
			gslbutils.SvcType, svcName))
	}

	routeList := acceptedRouteStore.GetAllClusterNSObjects()
	for _, routeName := range routeList {
		nodes.DequeueIngestion(gslbutils.MultiClusterKeyWithObjName(gslbutils.ObjectAdd,
			gslbutils.RouteType, routeName))
	}

	gslbutils.Logf("keys for GS graphs published to layer 3")

	sharedQ := utils.SharedWorkQueue().GetQueueByName(utils.GraphLayer)

	gsKeys := gsCache.AviCacheGetAllKeys()
	// find out the keys which are not already present in the list of created GS graphs
	agl := nodes.SharedAviGSGraphLister()
	dgl := nodes.SharedDeleteGSGraphLister()
	for _, gsKey := range gsKeys {
		key := gsKey.Tenant + "/" + gsKey.Name
		found, _ := agl.Get(key)
		if found {
			continue
		}
		gslbutils.Logf("key: %v, msg: didn't get a GS in the model cache", key)
		// create a new Graph with 0 members, push it to the delete queue
		newGSGraph := nodes.NewAviGSObjectGraph()
		newGSGraph.Name = gsKey.Name
		newGSGraph.Tenant = gsKey.Tenant
		newGSGraph.MemberObjs = []nodes.AviGSK8sObj{}
		newGSGraph.SetRetryCounter()
		dgl.Save(key, newGSGraph)

		bkt := utils.Bkt(key, sharedQ.NumWorkers)
		sharedQ.Workqueue[bkt].AddRateLimited(key)
		gslbutils.Logf("process: fullSync, modelName: %s, msg: %s", gsKey, "published key to rest layer")
	}

	// clean up any stale health monitors as well
	hmCache := avicache.GetAviHmCache()
	hmCacheKeys := hmCache.AviHmGetAllKeys()
	for _, hmKeyIntf := range hmCacheKeys {
		hmKey, ok := hmKeyIntf.(avicache.TenantName)
		if !ok {
			gslbutils.Debugf("key: %v, msg: hmKey object malformed", hmKey)
			continue
		}
		tenant, hmName := hmKey.Tenant, hmKey.Name
		gsName, err := gslbutils.GetGSFromHmName(hmName)
		if err != nil {
			gslbutils.Debugf("key: %v, msg: can't get gs name from hm", hmKey)
			continue
		}
		gsKey := tenant + "/" + gsName
		found, _ := agl.Get(gsKey)
		if found {
			continue
		}
		gslbutils.Logf("key: %v, msg: didn't get a GS in the model cache", gsKey)
		bkt := utils.Bkt(gsKey, sharedQ.NumWorkers)
		sharedQ.Workqueue[bkt].AddRateLimited(gsKey)
		gslbutils.Logf("process: fullSync, hmName: %s, modelName: %s, msg: published key to rest layer", hmName, gsName)
	}
}
