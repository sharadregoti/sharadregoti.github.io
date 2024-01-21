+++
title = 'Journey Of Deployment Creation In Kubernetes'
date = 2024-01-21T10:39:08+05:30
draft = false
+++

Hello everyone, This article is my take on what I have understood about the Kubernetes architecture.

Instead of explaining the different components that the architecture comprises of & what each component does functionally. 

We will take a Kubernetes feature & break its implementation to understand how it interacts with Kubernetes system components.

So yeah, some working knowledge of K8s basics such as creating deployments, and services is a plus.

Before starting, I want to let you know that this article is written to the best of my knowledge & I’ll be updating it as I gain more insights on it.

# The Initial Interaction With Kubernetes

My K8s journey started in 2020 when I was working as a developer at space-cloud (an open source tool which helps you to develop, deploy & secure your applications). At that time a decision was taken to support kubernetes for deployment & ditch docker support.

Being the founding engineer of space-cloud I was told to learn Kubernetes. So I started learning about the What, Why & How of Kubernetes.

I still remember my first hands on Kubernetes experience was to create a K8s cluster using minikube  in which I created a nginx deployment & exposed it using NodePort service. At the end of this exercise, I was able to view `ngnix` default web page running on `[localhost:8080](http://localhost:8080)`.

But still, Kubernetes was a black box for me. I didn’t know what is a deployment, what is a node port service, why Kubernetes is taking 8gb of RAM to run a nginx container & I bet it is a black box for a lot of people out there.

So we will dymistify this black box with an feature that everyone who is new to Kubernetes has seen it in action or at least heard about it.

> I am talking about deploying an application
> 

# Journey Of Deployment Resource

Here is an quick overview of the deployment journey represented via an image (for reading the image follow the numeric indicators highlighted in red color). This is what, we will be learning in depth in this article. Please keep this image in mind, we will be referring it a lot in this article.

!https://raw.githubusercontent.com/sharadregoti/try-out/master/01-kubernetes-the-hard-way/journey-of-deployment-resource-flow.svg

When you run `kubectl create deployment --image nginx` command on the terminal. Under the hood, kubectl fires an HTTP request to the server. And the server that responds to this request is called the `api-server`. Our first Kubernetes system component.

https://twitter.com/SharadRegoti/status/1562503866176147457?s=20&t=RhHkvAhY4j5Vs6jPayC3Gw

**API Server**

> A program that exposes the Kubernetes REST API for consumption & this component is also responsible for authentication & authorization of the entire system
> 

Following are the actions taken for every request be it external or internal

- Authenticates & authorizes the requests as per the configuration done in the `api-server`
- Depending upon the request, it performs CRUD operation on the persistent data store
- Sends the response back to the client

Apart from handling API requests it also handles operational activities of Kubernetes cluster such as

- Registration of worker nodes

In our case, As we have requested for creation of nginx deployment resource. After successful authentication the api-server will store the deployment resource object in the data store & sends back an appropriate HTTP status code to the client. 

If you observed carefully, the api-server talks to a persistent data store. It is our second component in the architecture `ETCD`

**ETCD Store**

> A distributed, consistent & highly available key-value store used for storing cluster data
> 

Nothing fancy here, Kubernetes is using ETCD as a database for storing the resource objects.

Apart from the basic CRUD operations that ETCD offers. A unique proposition of ETCD is providing events on changes happening to it’s keys. This feature is exposed by Kubernetes over the watch API. In the up coming sections you will see how different Kubernetes components leverages the watch API.

Till now, we haven’t taken any action with our nginx deployment object resting in the ETCD store. It’s time to do just that with our next component the `controller manager`

**Controller Manager**

> A program that takes actions on Kubernetes kinds submitted to the api-server
> 

Every kubernetes kind/resource (deployment, service etc…) needs to be understood & appropriate action has to be taken by some entity. That entity is called the `controller`. 

Technically, `controller` just watches for changes in a specific Kubernetes resource (deployment, service etc…) using the watch API exposed by `api-server` in a never ending for loop called the control loop. Whenever the controller is notified about resource change, It takes an appropriate action to ensure that current state of the resource matches with the desired state.

**How does a single `controller-manager` is able to handle multiple Kubernetes resource?**

The `controller-manager` binary comprises of many such resource `controllers` to take action on K8s resources. For example:

- Node controllers
- Deployment controller
- Service controller
- Namespace controller etc…

So in our case, As we created a new nginx deployment object. The deployment controller in the controller manager is notified about the new deployment object, the controller takes action by comparing the desired state (we specified it by our CLI command) & current state (what currently is running?). As there is no existing deployment, It takes the decision to create a new deployment.

Technically, a deployment in kubernetes is made of resources:

- **Pods:** Which is a logical grouping for running multiple containers.
- **Replica-Set:** Ensures that the replica (dublicates of pods) specified in the spec are running at any given time.

So the deployment controller tells the `api-server` to create replica-set & pod resource. Great!!!

As I said, that pod resource runs containers inside Kubernetes. But who decides on which node the container should run. As kuberentes is a multi-node system, someone has to decided where the container will run.

This is where the `scheduler` component comes into the picture.

**Scheduler**

> A program that watches for newly created pods with no assigned worker node & selects a ~~worker~~ node for them to run on
> 

But what really goes into the selection of ~~worker~~ node

There are many stages involved in the selection process, but the important once are,

- **Filtering**
    
    The goal of this state is to filter out the ~~worker~~ nodes that cannot host the pod for some reason, at the end of this stage we have nodes on which pods can be scheduled.
    
    Reasons for not hosting:
    
    - Taints & Tolerance
    - CPU & Memory requests
    - Node selector
    
    At the end of this stage, on the basis of length of schedulable node list following scenarios can occur:
    
    - Length equal to 0:  Then the pod will remain in the pending state till this condition is remedied.
    - Length equal to 1:  The scheduling can take place without any actions
    - Length greater than 1:  It moves to next stages of scheduling
    
- **Scoring**
    
    The goal of this stage is to assign scores to schedulable nodes on the basis of server rules. At the end of this stage the node with highest score is selected for scheduling of Pod
    
    Example of rules used for evaluation:
    
    - Node affinity & anti affinity
    - Does the node has container image already?
    - Lower workload utilization will give you higher result
    
    This is what a `scheduler` does at a high level.
    

Till now, We have only taken the decision of what to do & how to do it regarding our `nginx` deployment. But haven’t taken any action to actually run the container. 

**So a observation can be made that whatever components that we have seen acts as the brain of the cluster which has the decision-making capability of the entire cluster.**

In the kubernetes architecture, a separation is made between the decision making components called as **Master Node** & components which executes the decisions called as **Worker Node.**

!https://raw.githubusercontent.com/sharadregoti/try-out/master/01-kubernetes-the-hard-way/kubernetes-the-hard-way.drawio-Architecture.svg

!https://raw.githubusercontent.com/sharadregoti/try-out/master/01-kubernetes-the-hard-way/kubernetes-the-hard-way.drawio-Master%20Node.svg

As you have scene our `nginx` deployment has interacted with the master node in it’s journey. It’s time to go to the final destination

Ahh finally, we have some progress in running containers. So who runs our containers in the worker node? The answer is Kubelet.

**Kubelet**

> A program that watches for pods assigned to worker node (itself) & ensures the containers specified in the pod spec are running on the node
> 

Internally, kubelet uses many low level technologies for running containers for example:

- Container Runtime Interface
- Container Runtime (containerd, docker, podman etc…)
- Runc

All these low level technologies will require a separate article of their own, comment down below if you want a separate article on them. But here is an quick image for an overview.

!https://raw.githubusercontent.com/sharadregoti/try-out/master/01-kubernetes-the-hard-way/kubernetes-the-hard-way.drawio-Worker%20Node.svg

Finally, with the help of `kubelet` component our `nginx` container is running successfully. Husshh

# Ending Note

Generalized diagram of what we have learned…

!https://raw.githubusercontent.com/sharadregoti/try-out/master/01-kubernetes-the-hard-way/journey-of-deployment-resource-generalize.svg

We haven’t discussed 2 main Kubernetes components:

- **Cloud Control Manger**
- **Kube Proxy**

I’ll be writing a separate article on it, which will explain the journey of an HTTP request in Kubernetes!!!

That’s it in this blog post, If you liked this blog post. You can follow me on twitter @SharadRegoti where I talk about **Microservices, Cloud, Kubernetes & Golang.**