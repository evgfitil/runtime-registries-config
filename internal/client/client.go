package client

import (
	"context"
	"flag"
	"fmt"
	"github.com/caarlos0/env/v10"
	"github.com/evgfitil/runtime-registries-config/internal/logger"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"sort"
)

type Handlers interface {
	AddConfigMap(newData *[]ConfigMapData)
	UpdateConfigMap(newData *[]ConfigMapData)
	DeleteConfigMap(obj interface{})
}

type ConfigWatcher struct {
	ClientSet        *kubernetes.Clientset
	ConfigMapName    string `env:"CM_NAME" envDefault:"runtime-registry-config"`
	ConfigMapData    []ConfigMapData
	ConfigMapDataKey string `env:"CM_DATA_KEY" envDefault:"registries"`
	InClusterMode    bool   `env:"IN_CLUSTER" envDefault:"true"`
	Namespace        string `env:"NAMESPACE" envDefault:"default"`
}

type ConfigMapData struct {
	Original string `yaml:"original"`
	Mirror   string `yaml:"mirror"`
	Insecure bool   `yaml:"insecure"`
}

func NewConfigWatcher() (*ConfigWatcher, error) {
	watcher := &ConfigWatcher{ConfigMapData: make([]ConfigMapData, 0)}
	if err := env.Parse(watcher); err != nil {
		logger.Sugar.Fatalf("error reading environment variables: %v", err)
		return nil, err
	}

	clientSet, err := watcher.configureClient()
	if err != nil {
		logger.Sugar.Fatalf("error configuring kubernetes client: %v", err)
		return nil, err
	}

	watcher.ClientSet = clientSet
	return watcher, nil
}

func (cw *ConfigWatcher) GetConfigMapData() (*[]ConfigMapData, error) {
	configMap, err := cw.ClientSet.CoreV1().ConfigMaps(cw.Namespace).
		Get(context.Background(), cw.ConfigMapName, metav1.GetOptions{})
	if err != nil {
		logger.Sugar.Errorf("failed to get ConfigMap %s in namespace %s: %v", cw.ConfigMapName, cw.Namespace, err)
		return nil, err
	}

	data, ok := configMap.Data[cw.ConfigMapDataKey]
	if !ok {
		logger.Sugar.Errorf("configMap %s does not contain key %s", cw.ConfigMapName, cw.ConfigMapDataKey)
		return nil, err
	}
	var configMapData []ConfigMapData
	err = yaml.Unmarshal([]byte(data), &configMapData)
	if err != nil {
		logger.Sugar.Errorf("error unmarshaling configMap data: %v", err)
		return nil, err
	}
	cw.ConfigMapData = configMapData
	return &cw.ConfigMapData, nil
}

func (cw *ConfigWatcher) GetConfigMapDataFromObject(configMap *v1.ConfigMap) (*[]ConfigMapData, error) {
	if configMap == nil {
		return nil, fmt.Errorf("configMap is nil")
	}

	data, ok := configMap.Data[cw.ConfigMapDataKey]
	if !ok {
		return nil, fmt.Errorf("configMap %s does not contain key %s", configMap.Name, cw.ConfigMapDataKey)
	}

	var configMapData []ConfigMapData
	err := yaml.Unmarshal([]byte(data), &configMapData)
	if err != nil {
		logger.Sugar.Errorf("error unmarshaling configMap data: %v", err)
		return nil, err
	}

	return &configMapData, nil
}

func (cw *ConfigWatcher) TrackConfigChanges(handlers Handlers) {
	for {
		logger.Sugar.Infoln("start watching configMap update stream")
		listWatch := &cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				options.FieldSelector = fields.OneTermEqualSelector("metadata.name", cw.ConfigMapName).String()
				return cw.ClientSet.CoreV1().ConfigMaps(cw.Namespace).List(context.Background(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				options.FieldSelector = fields.OneTermEqualSelector("metadata.name", cw.ConfigMapName).String()
				return cw.ClientSet.CoreV1().ConfigMaps(cw.Namespace).Watch(context.Background(), options)
			},
		}

		_, controller := cache.NewInformer(
			listWatch,
			&v1.ConfigMap{},
			0,
			cache.ResourceEventHandlerFuncs{
				AddFunc: func(oldObj interface{}) {
					logger.Sugar.Infoln("added config map, starting update handler")
					newConfigMapData, err := cw.GetConfigMapData()
					if err != nil {
						logger.Sugar.Errorf("error retrieving configMap data: %v", err)
					}
					handlers.AddConfigMap(newConfigMapData)
				},
				UpdateFunc: func(oldObj, newObj interface{}) {
					logger.Sugar.Infoln("configMap is updated, starting update handler")
					oldConfigMap := oldObj.(*v1.ConfigMap)
					oldConfigMapData, err := cw.GetConfigMapDataFromObject(oldConfigMap)
					if err != nil {
						logger.Sugar.Errorf("error retrieving oldConfigMap data %v", err)
					}
					newConfigMapData, err := cw.GetConfigMapData()
					if err != nil {
						logger.Sugar.Errorf("error retrieving configMap data: %v", err)
					}
					if !compareConfigs(*oldConfigMapData, *newConfigMapData) {
						logger.Sugar.Infoln("data has changed, trigger update handler")
						handlers.UpdateConfigMap(newConfigMapData)
					} else {
						logger.Sugar.Infoln("data is unchanged")
					}
				},
				DeleteFunc: func(oldObj interface{}) {
					logger.Sugar.Infoln("configMap is deleted, starting delete handler")
					handlers.DeleteConfigMap(oldObj)
				},
			})

		stop := make(chan struct{})
		go controller.Run(stop)

		<-stop
	}
}

func (cw *ConfigWatcher) configureClient() (clientSet *kubernetes.Clientset, error error) {

	// out of cluster config
	if !cw.InClusterMode {
		var kubeconfig *string
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
		} else {
			kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
		}
		flag.Parse()

		config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			panic(err.Error())
		}

		clientSet, err = kubernetes.NewForConfig(config)
		if err != nil {
			panic(err.Error())
		}
		return clientSet, nil
	}

	// in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	clientSet, err = kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	return clientSet, nil
}

func compareConfigs(old, new []ConfigMapData) bool {
	sortConfigMapData(old)
	sortConfigMapData(new)

	if len(old) != len(new) {
		return false
	}

	for idx := range old {
		if old[idx].Original != new[idx].Original ||
			old[idx].Mirror != new[idx].Mirror ||
			old[idx].Insecure != new[idx].Insecure {
			return false
		}
	}
	return true
}

func sortConfigMapData(slice []ConfigMapData) {
	sort.Slice(slice, func(i, j int) bool {
		return slice[i].Original < slice[j].Original
	})
}
