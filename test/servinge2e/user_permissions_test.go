package servinge2e

import (
	"strings"
	"testing"

	"github.com/hemanrnjn/serverless-operator/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestUserPermissions(t *testing.T) {
	t.Run("user permissions", func(t *testing.T) {
		paCtx := test.SetupProjectAdmin(t)
		editCtx := test.SetupEdit(t)
		viewCtx := test.SetupView(t)
		test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, paCtx, editCtx, viewCtx) })
		defer test.CleanupAll(t, paCtx, editCtx, viewCtx)

		tests := []struct {
			name        string
			userContext *test.Context
			operation   func(context *test.Context) error
			wantErrStr  string
		}{{
			name: "user with view role can get",
			operation: func(c *test.Context) error {
				_, err := c.Clients.Serving.ServingV1().Services(testNamespace).Get(helloworldService, metav1.GetOptions{})
				return err
			},
			userContext: viewCtx,
		}, {
			name: "user with view role can list",
			operation: func(c *test.Context) error {
				_, err := c.Clients.Serving.ServingV1().Services(testNamespace).List(metav1.ListOptions{})
				return err
			},
			userContext: viewCtx,
		}, {
			name: "user with view role cannot create",
			operation: func(c *test.Context) error {
				_, err := test.CreateService(c, "userview-service", testNamespace, image)
				return err
			},
			userContext: viewCtx,
			wantErrStr:  "is forbidden",
		}, {
			name: "user with view role cannot delete",
			operation: func(c *test.Context) error {
				return c.Clients.Serving.ServingV1().Services(testNamespace).Delete(helloworldService, &metav1.DeleteOptions{})
			},
			userContext: viewCtx,
			wantErrStr:  "is forbidden",
		}, {
			name: "user with project admin role can get",
			operation: func(c *test.Context) error {
				_, err := c.Clients.Serving.ServingV1().Services(testNamespace).Get(helloworldService, metav1.GetOptions{})
				return err
			},
			userContext: paCtx,
		}, {
			name: "user with project admin role can list",
			operation: func(c *test.Context) error {
				_, err := c.Clients.Serving.ServingV1().Services(testNamespace).List(metav1.ListOptions{})
				return err
			},
			userContext: paCtx,
		}, {
			name: "user with project admin role can create",
			operation: func(c *test.Context) error {
				_, err := test.CreateService(c, "projectadmin-service", testNamespace, image)
				return err
			},
			userContext: paCtx,
		}, {
			name: "user with project admin role can delete",
			operation: func(c *test.Context) error {
				return c.Clients.Serving.ServingV1().Services(testNamespace).Delete("projectadmin-service", &metav1.DeleteOptions{})
			},
			userContext: paCtx,
		}, {
			name: "user with edit role can get",
			operation: func(c *test.Context) error {
				_, err := c.Clients.Serving.ServingV1().Services(testNamespace).Get(helloworldService, metav1.GetOptions{})
				return err
			},
			userContext: editCtx,
		}, {
			name: "user with edit role can list",
			operation: func(c *test.Context) error {
				_, err := c.Clients.Serving.ServingV1().Services(testNamespace).List(metav1.ListOptions{})
				return err
			},
			userContext: editCtx,
		}, {
			name: "user with edit role can create",
			operation: func(c *test.Context) error {
				_, err := test.CreateService(c, "useredit-service", testNamespace, image)
				return err
			},
			userContext: editCtx,
		}, {
			name: "user with edit role can delete",
			operation: func(c *test.Context) error {
				return c.Clients.Serving.ServingV1().Services(testNamespace).Delete("useredit-service", &metav1.DeleteOptions{})

			},
			userContext: editCtx,
		}}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				err := test.operation(test.userContext)
				if (err != nil) != (test.wantErrStr != "") {
					t.Errorf("User with role %s has unexpected behavior on knative services. Error thrown: %v, error expected: %t", test.userContext.Name, err, (test.wantErrStr != ""))
				}
				if err != nil && !strings.Contains(err.Error(), test.wantErrStr) {
					t.Errorf("Unexpected error for user with role %s: %v", test.userContext.Name, err)
				}
			})
		}
	})
}
