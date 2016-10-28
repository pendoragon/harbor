#### 1.build images
execute script：  
```sh
sh ./dockerfiles/build.sh
```

these images will be built(tags are decided by version in build.sh#4):
- harbor/ui:0.4.0  
- harbor/jobservice:0.4.0  
- harbor/mysql:0.4.0  
- harbor/registry:0.4.0  
- harbor/nginx:0.4.0  
 

dependencies:   
- Deploy/jsminify.sh  
- Deploy/db/registry.sql  
- Deploy/db/docker-entrypoint.sh  

#### 2.integrated with k8s
nginx ssl_certificate(need to set):
```
cert:./nginx/nginx.cm.yaml#104
pkey:./nginx/nginx.cm.yaml#108
```
you can find all env in *.rc.yaml and other configs in *.cm.yaml.    
most of configs have default value and can be changed.  



run in k8s:
```sh
#create pv & pvc
kubectl create -f ./pv/log.pv.yaml
kubectl create -f ./pv/registry.pv.yaml
kubectl create -f ./pv/storage.pv.yaml
kubectl create -f ./pv/log.pvc.yaml
kubectl create -f ./pv/registry.pvc.yaml
kubectl create -f ./pv/storage.pvc.yaml

#apply config map
kubectl apply -f ./jobservice/jobservice.cm.yaml
kubectl apply -f ./nginx/nginx.cm.yaml
kubectl apply -f ./registry/registry.cm.yaml
kubectl apply -f ./ui/ui.cm.yaml

#apply service
kubectl apply -f ./jobservice/jobservice.svc.yaml
kubectl apply -f ./mysql/mysql.svc.yaml
kubectl apply -f ./nginx/nginx.svc.yaml
kubectl apply -f ./registry/registry.svc.yaml
kubectl apply -f ./ui/ui.svc.yaml

#create k8s rc
kubectl create -f ./registry/registry.rc.yaml
kubectl create -f ./mysql/mysql.rc.yaml
kubectl create -f ./jobservice/jobservice.rc.yaml
kubectl create -f ./ui/ui.rc.yaml
kubectl create -f ./nginx/nginx.rc.yaml
```

