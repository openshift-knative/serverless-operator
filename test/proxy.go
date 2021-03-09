package test

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func UpdateGlobalProxy(ctx *Context, value string) error {
	proxy, err := ctx.Clients.ConfigClient.Proxies().Get(context.Background(), "cluster", metav1.GetOptions{})
	if err != nil {
		return err
	}
	proxy.Spec.HTTPProxy = value
	if _, err := ctx.Clients.ConfigClient.Proxies().Update(context.Background(), proxy, metav1.UpdateOptions{}); err != nil {
		return err
	}
	return nil
}
