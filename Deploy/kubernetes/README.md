
# Integration with Kubernetes

Firstly you need to get docker images of harbor.You can get it by the two ways:
- Download Harbor's images from Docker hub. See [Installation Guide](https://github.com/vmware/harbor/blob/master/docs/installation_guide.md)
- Build images by **dockerfiles/build.sh**(For testing quickly,**DO NOT** use it in production)

Use **dockerfiles/build.sh** to build images:  
```
bash ./dockerfiles/build.sh [version]
```


**build.sh** has two optional parameters:
- version($1):version is the tag of images.Default value is 'latest'
- nopull($2):nopull prevents script from pulling nginx and registry.Default value is 'false'


These images will be built or pulled(Tags are decided by version):
- harbor/ui:version  
- harbor/jobservice:version  
- harbor/mysql:version  
- harbor/registry:version  
- harbor/nginx:version  

You can put these images into all kubernetes minions or a docker registry where your cluster can pull them.    

Building dependencies:   
- Deploy/jsminify.sh  
- Deploy/db/registry.sql  
- Deploy/db/docker-entrypoint.sh  


Before you starting harbor in kubernetes,you need to create some configs with these steps:  
1. Set configs in harbor.cfg  
2. Prepare your https pkey and cert  
3. Make sure there is a version of openssl in your environment  
4. Execute ./prepare:  
```
# generates *.cm.yaml automatically 
python3 ./prepare -k path-to-https-pkey -c path-to-https-cert
```

./prepare generates *.cm.yaml from templates. There are some key configs in *.cm.yaml:
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
  - use other storage ways instead of hostPath(**!!!IMPORTENT**)

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