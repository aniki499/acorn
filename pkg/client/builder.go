package client

import (
	"context"
	"net/http"
	"time"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/watcher"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (c *client) getOrCreateBuilder(ctx context.Context, name string) (*apiv1.Builder, error) {
	builder := &apiv1.Builder{}
	if name == "" {
		builders := &apiv1.BuilderList{}
		if err := c.Client.List(ctx, builders, &kclient.ListOptions{Namespace: c.Namespace}); err != nil {
			return nil, err
		}

		if len(builders.Items) == 0 {
			builder = &apiv1.Builder{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "default",
					Namespace: c.Namespace,
				},
			}
			if err := c.Client.Create(ctx, builder); err != nil {
				return nil, err
			}
		} else {
			builder = &builders.Items[0]
		}
	} else {
		if err := c.Client.Get(ctx, router.Key(c.Namespace, name), builder); err != nil {
			return nil, err
		}
	}

	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		select {
		case <-subCtx.Done():
		case <-time.After(3 * time.Second):
			logrus.Infof("Waiting for builder to start")
		}
	}()

	builder, err := watcher.New[*apiv1.Builder](c.Client).ByObject(ctx, builder, func(builder *apiv1.Builder) (bool, error) {
		return builder.Status.Ready, nil
	})
	if err != nil {
		return nil, err
	}

	if builder.Status.Ready && builder.Status.Endpoint != "" {
		for i := 0; i < 5; i++ {
			resp, err := http.Get(builder.Status.Endpoint + "/ping")
			if err != nil {
				logrus.Debugf("builder ping failed: %v", err)
			} else {
				_ = resp.Body.Close()
				logrus.Debugf("builder status code: %d", resp.StatusCode)
				if resp.StatusCode == http.StatusOK {
					return builder, nil
				}
			}
			time.Sleep(time.Second)
		}
	}

	return builder, nil
}
