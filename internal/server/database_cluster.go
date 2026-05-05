// everest
// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package server contains the API server implementation.
package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
	"github.com/percona/everest/api"
)

var (
	errFailedToGetUser         = errors.New("failed to get user from context")
	errFailedToReadRequestBody = errors.New("failed to read request body")
)

// CreateDatabaseCluster creates a new db cluster inside the given k8s cluster.
func (e *EverestServer) CreateDatabaseCluster(c echo.Context, namespace string) error {
	dbc := &everestv1alpha1.DatabaseCluster{}
	if err := e.getBodyFromContext(c, dbc); err != nil {
		return errors.Join(errFailedToReadRequestBody, err)
	}
	dbc.SetNamespace(namespace)

	result, err := e.handler.CreateDatabaseCluster(c.Request().Context(), dbc)
	if err != nil {
		e.l.Errorf("CreateDatabaseCluster failed: %v", err)
		return err
	}

	// Collect metrics immediately after a DB cluster has been created.
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		if err := e.collectMetrics(ctx, *e.config); err != nil {
			e.l.Errorf("Could not send metrics: %v", err)
		}
	}()
	return c.JSON(http.StatusCreated, result)
}

// ListDatabaseClusters lists the created database clusters on the specified kubernetes cluster.
func (e *EverestServer) ListDatabaseClusters(ctx echo.Context, namespace string) error {
	list, err := e.handler.ListDatabaseClusters(ctx.Request().Context(), namespace)
	if err != nil {
		e.l.Errorf("ListDatabaseClusters failed: %v", err)
		return err
	}
	return ctx.JSON(http.StatusOK, list)
}

// DeleteDatabaseCluster deletes a database cluster on the specified kubernetes cluster.
func (e *EverestServer) DeleteDatabaseCluster(
	c echo.Context,
	namespace, name string,
	params api.DeleteDatabaseClusterParams,
) error {
	if err := e.handler.DeleteDatabaseCluster(c.Request().Context(), namespace, name, &params); err != nil {
		e.l.Errorf("DeleteDatabaseCluster failed: %v", err)
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// GetDatabaseCluster retrieves the specified database cluster on the specified kubernetes cluster.
func (e *EverestServer) GetDatabaseCluster(c echo.Context, namespace, name string) error {
	result, err := e.handler.GetDatabaseCluster(c.Request().Context(), namespace, name)
	if err != nil {
		e.l.Errorf("GetDatabaseCluster failed: %v", err)
		return err
	}
	return c.JSON(http.StatusOK, result)
}

// GetDatabaseClusterComponents returns database cluster components.
func (e *EverestServer) GetDatabaseClusterComponents(c echo.Context, namespace, name string) error {
	result, err := e.handler.GetDatabaseClusterComponents(c.Request().Context(), namespace, name)
	if err != nil {
		e.l.Errorf("GetDatabaseClusterComponents failed: %v", err)
		return err
	}
	return c.JSON(http.StatusOK, result)
}

func (e *EverestServer) GetDatabaseClusterComponentLogs(c echo.Context, ns, cName, componentName string, params api.GetDatabaseClusterComponentLogsParams) error {
	ctx := c.Request().Context()

	// function to stream logs. it uses closures for the echo-related dependencies to keep the handlers (validation, rbac, k8s) independent from http-framework
	stream := func(ctx context.Context, namespace, clusterName, componentName string, params api.GetDatabaseClusterComponentLogsParams) error {
		opts, err := e.buildPodLogOptions(ctx, namespace, componentName, params)
		if err != nil {
			return err
		}

		req := e.kubeStreamer.CoreV1().Pods(namespace).GetLogs(componentName, opts)
		stream, err := req.Stream(ctx)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadGateway, "failed to open log stream: "+err.Error())
		}
		defer stream.Close()

		return streamToResponse(ctx, c.Response(), stream)
	}

	err := e.handler.GetDatabaseClusterComponentLogs(ctx, ns, cName, componentName, params, stream)
	if err != nil {
		e.l.Errorf("GetDatabaseClusterComponents failed: %v", err)
		return err
	}
	return nil
}

func (e *EverestServer) buildPodLogOptions(
	ctx context.Context,
	namespace, componentName string,
	params api.GetDatabaseClusterComponentLogsParams,
) (*corev1.PodLogOptions, error) {
	const (
		defaultTailLines int64 = 200
		maxLimitBytes          = 100 * 1024 * 1024 // 100 MiB
	)

	follow := params.Follow != nil && *params.Follow

	opts := &corev1.PodLogOptions{
		Follow: follow,
		Container: e.getDefaultContainer(
			ctx,
			namespace,
			componentName,
			params.Container,
		),
	}

	if params.TailLines != nil {
		opts.TailLines = intPtrToInt64Ptr(params.TailLines)
	} else if !follow {
		opts.TailLines = ptr.To(defaultTailLines)
	}

	if params.SinceSeconds != nil {
		opts.SinceSeconds = intPtrToInt64Ptr(params.SinceSeconds)
	}

	if params.SinceTime != nil {
		opts.SinceTime = &metav1.Time{Time: *params.SinceTime}
	}

	if params.Previous != nil {
		opts.Previous = *params.Previous
	}

	if params.LimitBytes != nil {
		if *params.LimitBytes > maxLimitBytes {
			return nil, errors.New("limitBytes too large")
		}
		opts.LimitBytes = intPtrToInt64Ptr(params.LimitBytes)
	}

	if params.Timestamps != nil {
		opts.Timestamps = *params.Timestamps
	}

	return opts, nil
}

func streamToResponse(
	ctx context.Context,
	res *echo.Response,
	stream io.Reader,
) error {
	res.Header().Set(echo.HeaderContentType, "text/plain; charset=utf-8")
	res.Header().Set("Transfer-Encoding", "chunked")
	res.WriteHeader(http.StatusOK)

	flusher, ok := res.Writer.(http.Flusher)
	if !ok {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"streaming not supported",
		)
	}

	buf := make([]byte, 32*1024)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			n, err := stream.Read(buf)
			if n > 0 {
				if _, werr := res.Write(buf[:n]); werr != nil {
					return nil // client disconnected
				}
				flusher.Flush()
			}
			if err != nil {
				if errors.Is(err, io.EOF) {
					return nil
				}
				return err
			}
		}
	}
}

// by default the first container in the list is used to get the logs from.
func (e *EverestServer) getDefaultContainer(ctx context.Context, namespace, podName string, componentName *string) string {
	if componentName != nil {
		return *componentName
	}
	pods, err := e.kubeConnector.ListPods(ctx, ctrlclient.InNamespace(namespace), ctrlclient.MatchingFields{"metadata.name": podName})
	if err != nil {
		e.l.Errorf("ListPods failed: %v", err)
		return ""
	}
	if len(pods.Items) == 0 {
		e.l.Errorf("Empty pod list")
		return ""
	}
	if len(pods.Items[0].Spec.Containers) == 0 {
		e.l.Errorf("Empty container list")
		return ""
	}

	return pods.Items[0].Spec.Containers[0].Name
}

func intPtrToInt64Ptr(v *int) *int64 {
	if v == nil {
		return nil
	}
	x := int64(*v)
	return &x
}

// UpdateDatabaseCluster replaces the specified database cluster on the specified kubernetes cluster.
//
//nolint:dupl
func (e *EverestServer) UpdateDatabaseCluster(ctx echo.Context, namespace, name string) error {
	dbc := &everestv1alpha1.DatabaseCluster{}
	if err := e.getBodyFromContext(ctx, dbc); err != nil {
		return errors.Join(errFailedToReadRequestBody, err)
	}
	dbc.SetNamespace(namespace)
	dbc.SetName(name)

	result, err := e.handler.UpdateDatabaseCluster(ctx.Request().Context(), dbc)
	if err != nil {
		e.l.Errorf("UpdateDatabaseCluster failed: %v", err)
		return err
	}
	return ctx.JSON(http.StatusOK, result)
}

// GetDatabaseClusterCredentials returns credentials for the specified database cluster.
func (e *EverestServer) GetDatabaseClusterCredentials(c echo.Context, namespace, name string) error {
	result, err := e.handler.GetDatabaseClusterCredentials(c.Request().Context(), namespace, name)
	if err != nil {
		e.l.Errorf("GetDatabaseClusterCredentials failed: %v", err)
		return err
	}
	return c.JSON(http.StatusOK, result)
}

// GetDatabaseClusterPitr returns the point-in-time recovery related information for the specified database cluster.
func (e *EverestServer) GetDatabaseClusterPitr(c echo.Context, namespace, name string) error {
	result, err := e.handler.GetDatabaseClusterPitr(c.Request().Context(), namespace, name)
	if err != nil {
		e.l.Errorf("GetDatabaseClusterPitr failed: %v", err)
		return err
	}
	return c.JSON(http.StatusOK, result)
}

// CreateDatabaseClusterSecret creates a secret for the specified database cluster.
func (e *EverestServer) CreateDatabaseClusterSecret(
	c echo.Context,
	namespace,
	dbName string,
	params api.CreateDatabaseClusterSecretParams,
) error {
	secret := &corev1.Secret{}
	if err := e.getBodyFromContext(c, secret); err != nil {
		return errors.Join(errFailedToReadRequestBody, err)
	}
	secret.SetNamespace(namespace)

	// if name is not set, generate a random one
	name := secret.GetName()
	if name == "" {
		randNum := rand.Intn(90000) + 10000 //nolint:gosec
		name = fmt.Sprintf("%s-%d-secret", dbName, randNum)
		secret.SetName(name)
	}
	result, err := e.handler.CreateDatabaseClusterSecret(c.Request().Context(), namespace, dbName, secret)
	if err != nil {
		e.l.Errorf("CreateDatabaseClusterSecret failed: %v", err)
		return err
	}
	return c.JSON(http.StatusCreated, result)
}
