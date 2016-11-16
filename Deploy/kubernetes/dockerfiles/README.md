
## Guide for Building Images of Harbor
This guide show you how to build images of Harbor.

### Prerequisite
Before you running Harbor, you need 5 images:
```
harbor/ui  
harbor/jobservice  
harbor/mysql  
harbor/registry  
harbor/nginx  
```

Make sure that you have these files in corrrect position:
- Deploy/jsminify.sh  
- Deploy/db/registry.sql  
- Deploy/db/docker-entrypoint.sh 


### Build Images
In the 5 images, We should build the first 3 images and pull others from Docker Hub.
 For your convenience, We provide a shell script `build.sh` to do these work.

Use `build.sh` to build images:  
```
bash ./build.sh
```
When the script finished its work, you can get these images:
```
harbor/ui:latest  
harbor/jobservice:latest  
harbor/mysql:latest  
harbor/registry:latest  
harbor/nginx:latest  
```
Default tag is latest. If you want a specified tag, you should build with:
```
bash ./build.sh tag-name
```


The script will pull `registry` and `nginx` automatically.
 If you don't need these images or you want a specified version of the two images,
 you can use `nopull` ( In this occasion, you must set the `tag-name` ):
```
bash ./build.sh tag-name nopull
```

