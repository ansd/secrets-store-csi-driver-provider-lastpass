package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/ansd/lastpass-go"
	"gopkg.in/yaml.v2"
	"k8s.io/klog/v2"
	"sigs.k8s.io/secrets-store-csi-driver/provider/v1alpha1"
)

type CSIDriverProviderServer struct{}

type mountConfig struct {
	podInfo     *podInfo
	auth        *auth
	items       []*item
	output      string
	permissions os.FileMode
}

// podInfo includes details about the pod that is receiving the mount event
type podInfo struct {
	namespace      string
	name           string
	serviceAccount string
}

// auth contains LastPass login credentials
type auth struct {
	username       string
	masterPassword string
}

// Item holds the parameters of the SecretProviderClass CRD.
type item struct {
	// Shared folder of the LastPass item.
	Share string `json:"share" yaml:"share"`
	// Group of the LastPass item.
	Group string `json:"group" yaml:"group"`
	// Name of the LastPass item.
	Name string `json:"name" yaml:"name"`
}

func (s *CSIDriverProviderServer) Version(ctx context.Context, req *v1alpha1.VersionRequest) (*v1alpha1.VersionResponse, error) {
	klog.Info("received version request")
	return &v1alpha1.VersionResponse{
		Version:        "v1alpha1",
		RuntimeName:    "secrets-store-csi-driver-provider-lastpass",
		RuntimeVersion: "v0.1.1",
	}, nil
}

func (s *CSIDriverProviderServer) Mount(ctx context.Context, req *v1alpha1.MountRequest) (*v1alpha1.MountResponse, error) {
	klog.Info("received mount request")
	mountConfig, err := parse(req)
	if err != nil {
		return &v1alpha1.MountResponse{}, err
	}
	klog.InfoS("parsed mount config",
		"pod", klog.ObjectRef{
			Namespace: mountConfig.podInfo.namespace,
			Name:      mountConfig.podInfo.name})

	return mount(ctx, mountConfig)
}

func parse(req *v1alpha1.MountRequest) (*mountConfig, error) {
	var secrets, attr map[string]string
	var filePermission os.FileMode
	mountConfig := &mountConfig{}

	if err := json.Unmarshal([]byte(req.GetPermission()), &filePermission); err != nil {
		klog.ErrorS(err, "failed to unmarshal file permission")
		return mountConfig, fmt.Errorf("failed to unmarshal file permission, error: %w", err)
	}
	mountConfig.permissions = filePermission

	if err := json.Unmarshal([]byte(req.GetSecrets()), &secrets); err != nil {
		klog.ErrorS(err, "failed to unmarshal node publish secrets ref")
		return mountConfig, fmt.Errorf("failed to unmarshal secrets, error: %w", err)
	}
	if secrets == nil {
		return mountConfig, fmt.Errorf("failed to get LastPass credentials, nodePublishSecretRef secret is not set")
	}
	var username, passwd string
	for k, v := range secrets {
		switch strings.ToLower(k) {
		case "username":
			username = v
		case "password":
			passwd = v
		}
	}
	if username == "" {
		return mountConfig, errors.New("could not find username in secrets")
	}
	if passwd == "" {
		return mountConfig, errors.New("could not find password in secrets")
	}
	mountConfig.auth = &auth{
		username:       username,
		masterPassword: passwd,
	}

	if err := json.Unmarshal([]byte(req.GetAttributes()), &attr); err != nil {
		klog.ErrorS(err, "failed to unmarshal attributes")
		return mountConfig, fmt.Errorf("failed to unmarshal attributes, error: %w", err)
	}

	mountConfig.podInfo = &podInfo{
		namespace:      attr["csi.storage.k8s.io/pod.namespace"],
		name:           attr["csi.storage.k8s.io/pod.name"],
		serviceAccount: attr["csi.storage.k8s.io/serviceAccount.name"],
	}

	if _, ok := attr["items"]; !ok {
		return mountConfig, errors.New("missing required 'items' attribute")
	}
	if err := yaml.Unmarshal([]byte(attr["items"]), &mountConfig.items); err != nil {
		return mountConfig, fmt.Errorf("failed to unmarshal 'items' attribute: %v", err)
	}

	mountConfig.output = attr["output"]

	return mountConfig, nil
}

func mount(ctx context.Context, mountConfig *mountConfig) (*v1alpha1.MountResponse, error) {
	lpassClient, err := lastpass.NewClient(ctx, mountConfig.auth.username, mountConfig.auth.masterPassword)
	if err != nil {
		klog.ErrorS(err, "failed to authenticate with LastPass server", "username", mountConfig.auth.username)
		return &v1alpha1.MountResponse{}, fmt.Errorf("failed to authenticate with LastPass server, error: %w", err)
	}

	lastPassAccounts, err := lpassClient.Accounts(ctx)
	if err != nil {
		klog.ErrorS(err, "failed to read LastPass accounts", "username", mountConfig.auth.username)
		return &v1alpha1.MountResponse{}, fmt.Errorf("failed to read LastPass accounts, error: %w", err)
	}

	if err := lpassClient.Logout(ctx); err != nil {
		klog.ErrorS(err, "failed to logout from LastPass", "username", mountConfig.auth.username)
		return &v1alpha1.MountResponse{}, fmt.Errorf("failed to logout from LastPass, error: %w", err)
	}

	files := []*v1alpha1.File{}
	ovs := []*v1alpha1.ObjectVersion{}

	for _, item := range mountConfig.items {
		for _, acct := range lastPassAccounts {
			if acct.Share == item.Share && acct.Group == item.Group && acct.Name == item.Name {
				var contents []byte
				if mountConfig.output == "" {
					// by default output account's JSON representation
					contents, err = json.Marshal(acct)
					if err != nil {
						return &v1alpha1.MountResponse{}, err
					}
				} else {
					v := reflect.ValueOf(acct)
					f := reflect.Indirect(v).FieldByName(mountConfig.output)
					contents = []byte(f.String())
				}
				path := filepath.Join(acct.Share, acct.Group, acct.Name)
				files = append(files, &v1alpha1.File{
					Path:     path,
					Contents: contents,
					Mode:     int32(mountConfig.permissions),
				})
				ovs = append(ovs, &v1alpha1.ObjectVersion{
					Id:      acct.ID,
					Version: acct.LastModifiedGMT,
				})
				klog.InfoS("added LastPass item to response",
					"path", path,
					"lastpass_item_id", acct.ID,
					"pod", klog.ObjectRef{Namespace: mountConfig.podInfo.namespace, Name: mountConfig.podInfo.name})
				break
			}
		}
	}

	if len(files) != len(mountConfig.items) {
		klog.Warningf("requested %d LastPass items, but found only %d matched items", len(mountConfig.items), len(files))
	}

	return &v1alpha1.MountResponse{
		ObjectVersion: ovs,
		Files:         files,
	}, nil
}
