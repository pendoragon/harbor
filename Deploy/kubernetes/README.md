
## Integration with Kubernetes
This Document decribes how to deploy Harbor on Kubernetes.

### Prerequisite
You need to get docker images of Harbor. You can get it by the two ways:
- Download Harbor's images from Docker hub. See [Installation Guide](https://github.com/vmware/harbor/blob/master/docs/installation_guide.md)
- Build images by `build.sh` . See [Guild for Building Images of Harbor](./dockerfiles/README.md)


### Configuration
We provide a python script `./prepare` to generate Kubernetes ConfigMap files. 
 The script is written in python3, so you need a python in your environment which version is greater than 3.
 Also the script need `openssl` to generate private key and certification, make sure you have a version of `openssl` in your environment. 

There are some args of the script:
- -f: Default Value is `../harbor.cfg` . You can specify other config file of Harbor.
- -k: Path to https private key. If you want to use https in Harbor, you should set it
- -c: Path to https certification. If you want to use https in Harbor, you should set it 

#### Basic Configuration
These Basic Configuration must be set. Otherwise you can't deploy Harbor on Kubernetes.
- `harbor.cfg` : Basic config of Harbor. Please refer to `harbor.cfg` .
- `*.pvc.yaml` : Persistent Volume Claim.  
  You can set capacity of storage in these files. example:
  
  ```
  resources:
    requests:
      # you can set another value to adapt to your needs
      storage: 100Gi
  ```
  
- `*.pv.yaml` : Persistent Volume. Be bound with `*.pvc.yaml` .  
  PVs and PVCs are one to one correspondence. If you changed capacity of PVC, you need to set capacity of PV together.
  example:
  
  ```
  capacity:
    # same value with PVC
    storage: 100Gi
  ```
  
  In PV, you should set another way to store data rather than `hostPath`:
  
  ```
  # it's default value, you should use others like nfs.
  hostPath:
    path: /data/registry
  ```
  
  For more infomation about store ways, Please check [Kubernetes Document](http://kubernetes.io/docs/user-guide/persistent-volumes/) 

Then you can generate ConfigMap files by :

```
python3 ./prepare -f ../harbor.cfg -k path-to-https-pkey -c path-to-https-cert
```

These files will be generated:
- ./jobservice/jobservice.cm.yaml
- ./mysql/mysql.cm.yaml
- ./nginx/nginx.cm.yaml
- ./registry/registry.cm.yaml
- ./ui/ui.cm.yaml

#### Advanced Configuration
If Basic Configuration was not covering your requirements, you can read this section for more details.

`./prepare` has a specify format of placeholder:
- `{{key}}` : It means we should replace the placeholder with the value in `config.cfg` which name is `key` .
- `{{num key}}` : It's used for multiple lines text. It will add `num` spaces to the leading of every line in text.

You can find all configs of Harbor in `./templates/`.There are specifications of these files:
- `jobservice.cm.yaml` : ENV and web config of jobservice
- `mysql.cm.yaml` : Root passowrd of MySQL
- `nginx.cm.yaml` : Https certification and nginx config  
  If you are fimiliar with nginx, you can modify it. 
- `registry.cm.yaml` : Token service certification and registry config
  Registry use filesystem to store data of images. You can find it like:
  
  ```
  storage:
      filesystem:
        rootdirectory: /storage
  ``` 
  
  If you want use another storage backend, please see [Docker Doc](https://docs.docker.com/datacenter/dtr/2.1/guides/configure/configure-storage/)
- `ui.cm.yaml` : Token service private key, ENV and web config of ui 

`ui` and `jobservice` are powered by beego. If you are fimiliar with beego, you can modify configs in `jobservice.cm.yaml` and `ui.cm.yaml` .




### Running
When you finished your configuring and generated ConfigMap files, you can run Harbor on kubernetes with these commands:
```
# create pv & pvc
kubectl apply -f ./pv/log.pv.yaml
kubectl apply -f ./pv/registry.pv.yaml
kubectl apply -f ./pv/storage.pv.yaml
kubectl apply -f ./pv/log.pvc.yaml
kubectl apply -f ./pv/registry.pvc.yaml
kubectl apply -f ./pv/storage.pvc.yaml

# create config map
kubectl apply -f ./jobservice/jobservice.cm.yaml
kubectl apply -f ./mysql/mysql.cm.yaml
kubectl apply -f ./nginx/nginx.cm.yaml
kubectl apply -f ./registry/registry.cm.yaml
kubectl apply -f ./ui/ui.cm.yaml

# create service
kubectl apply -f ./jobservice/jobservice.svc.yaml
kubectl apply -f ./mysql/mysql.svc.yaml
kubectl apply -f ./nginx/nginx.svc.yaml
kubectl apply -f ./registry/registry.svc.yaml
kubectl apply -f ./ui/ui.svc.yaml

# create k8s rc
kubectl apply -f ./registry/registry.rc.yaml
kubectl apply -f ./mysql/mysql.rc.yaml
kubectl apply -f ./jobservice/jobservice.rc.yaml
kubectl apply -f ./ui/ui.rc.yaml
kubectl apply -f ./nginx/nginx.rc.yaml

```
