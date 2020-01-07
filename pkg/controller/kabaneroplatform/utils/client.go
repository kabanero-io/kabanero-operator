package utils

import (
	"context"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Wrapper around the controller-runtime client update method to allow
// create and update to share the same signature
func Update(c client.Client, ctx context.Context, obj runtime.Object) error {
	return c.Update(ctx, obj)
}

func Create(c client.Client, ctx context.Context, obj runtime.Object) error {
	return c.Create(ctx, obj)
}
