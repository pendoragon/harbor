
# Integration with Kubernetes

Firstly you need to get docker images of harbor.You can get it by the two ways:
- Download Harbor's images from Docker hub. see [Installation guide](https://github.com/vmware/harbor/blob/master/docs/installation_guide.md)
- Build images by **dockerfiles/build.sh**

Use **dockerfiles/build.sh** to build images:  
```
bash ./dockerfiles/build.sh [version]
```


**build.sh** has two optional parameters:
- version($1):version is the tag of images.Default value is 'latest'
- nopull($2):nopull prevents script from pulling nginx and registry.Default value is 'false'


These images will be built or pulled(tags are decided by version):
- harbor/ui:version  
- harbor/jobservice:version  
- harbor/mysql:version  
- harbor/registry:version  
- harbor/nginx:version  

You can put these images into all kubernetes minions or a docker registry which your cluster can find it.    

Building dependencies:   
- Deploy/jsminify.sh  
- Deploy/db/registry.sql  
- Deploy/db/docker-entrypoint.sh  


Before you starting harbor in kubernetes,you need to set some configs:
- mysql/mysql.cm.yaml:
  - password of root
- registry/registry.cm.yaml:
  - storage of registry
  - auth
  - cert of auth token
- ui/ui.cm.yaml:
  - password of mysql
  - password of admin
  - ui secret
  - secret key 
  - auth mode
  - web config
  - private key of auth token
- jobservice/jobservice.cm.yaml:
  - password of mysql
  - ui secret
  - secret key
  - web config
- nginx/nginx.cm.yaml:
  - nginx config 
  - private key of https
  - cert of https
- pv/*.pv.yaml
  - use other storage ways instead of hostPath

Then you can start harbor in kubernetes with these commands:
```
# create pv & pvc
kubectl apply -f ./pv/log.pv.yaml
kubectl apply -f ./pv/registry.pv.yaml
kubectl apply -f ./pv/storage.pv.yaml
kubectl apply -f ./pv/log.pvc.yaml
kubectl apply -f ./pv/registry.pvc.yaml
kubectl apply -f ./pv/storage.pvc.yaml

# apply config map
kubectl apply -f ./jobservice/jobservice.cm.yaml
kubectl apply -f ./mysql/mysql.cm.yaml
kubectl apply -f ./nginx/nginx.cm.yaml
kubectl apply -f ./registry/registry.cm.yaml
kubectl apply -f ./ui/ui.cm.yaml

# apply service
kubectl apply -f ./jobservice/jobservice.svc.yaml
kubectl apply -f ./mysql/mysql.svc.yaml
kubectl apply -f ./nginx/nginx.svc.yaml
kubectl apply -f ./registry/registry.svc.yaml
kubectl apply -f ./ui/ui.svc.yaml

# create k8s rc
kubectl create -f ./registry/registry.rc.yaml
kubectl create -f ./mysql/mysql.rc.yaml
kubectl create -f ./jobservice/jobservice.rc.yaml
kubectl create -f ./ui/ui.rc.yaml
kubectl create -f ./nginx/nginx.rc.yaml

```