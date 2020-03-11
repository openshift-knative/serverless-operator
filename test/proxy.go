package test

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func UpdateGlobalProxy(ctx *Context, value string) error {
	proxy, err := ctx.Clients.ProxyConfig.Proxies().Get("cluster", metav1.GetOptions{})
	if err != nil {
		return err
	}
	proxy.Spec.HTTPProxy = value
	if _, err := ctx.Clients.ProxyConfig.Proxies().Update(proxy); err != nil {
		return err
	}
	return nil
}
