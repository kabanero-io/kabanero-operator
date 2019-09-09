
## Building an image locally

The Makefile understands the IMAGE environment variable. This can be provided when executing the `build-image` makefile target:

```
IMAGE=myrepo/kabanero-operator:test build-image
...
docker push myrepo/kabanero-operator:test
```


## Building and Deploying

```


```


## Common Workflows

### Build and Install a CRD modification

```
make generate
make install
```

### Run the operator in a terminal session
```
# Initial install of the CRDs and deploy other dependencies
make install
make deploy
# Remove the operator deployment since we will run this in the terminal
kubectl -n kabanero delete deployment kabanero-operator 

# Then just run this as needed
operator-sdk up local
```

### Build and deploy an image
```
IMAGE=myrepo/kabanero-operator:test build-image
docker push myrepo/kabanero-operator:test
IMAGE=myrepo/kabanero-operator:test deploy
```

