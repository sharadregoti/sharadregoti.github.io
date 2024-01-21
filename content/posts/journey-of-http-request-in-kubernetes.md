+++
title = 'Journey Of HTTP Request In Kubernetes'
date = 2024-01-21T10:39:08+05:30
draft = false
+++

In the previous article, we have learnt the journey of deployment creation in Kubernetes. Which helped us understand almost every aspect of the core system components of Kubernetes, but it missed the following components:

- Cloud Control Manager
- Kube Proxy

To understand the above components, we will use the same technique of previous article where we will take a Kubernetes feature & break its implementation to understand how it interacts with the Kubernetes system components.

The feature for this article is going to be:

> Exposing an application using the service type load balancer
> 

I highly recommend you read the previous article, I’ll be referring some concepts from that article.

Having understanding of the Service Resource of Kubernetes is a pre-requisite for this article. If you don’t know what is a service type resource, No worries! Checkout this article which explains the What, Why & How of Kubernetes services.

Before starting, I want to let you know that this article is written to the best of my knowledge & I’ll be updating it as I gain more insights on it.

Throughout this post, I’ll reference

- Cloud Controller Manager as “CCM”
- Kube Controller Manager as “KCM”
- Load Balancer as “LB”
- Kubernetes as “k8s”

Let’s start our article with a question

# What Happens After Deployment?

So you have deployed your application on Kubernetes, That’s great!!!

But now, how will your users access this application? What I meant by access is, when the users type your domain name on the browser how that domain will resolve to the IP address of the container residing in your Kubernetes cluster?

Don’t have the answer for it? No worries! That’s exactly, what we are going to learn in this article.

For the domain name to the container IP address resolution to work, we need to expose your application to the outside world & that is where the service type Load Balancer of Kubernetes comes into the picture.

When you use this resource, It magically exposes your application on the internet & provides you with a static external IP address that can be mapped to your domain name (using something like godaddy).

As per the below diagram, when your users type the domain name in the browser, it resolves to the external address provided by the Load Balancer service which then redirects the request to the IP address of container residing in Kubernetes.

!https://raw.githubusercontent.com/sharadregoti/try-out/master/01-kubernetes-the-hard-way/journey-of-http-request-load-balancer.drawio.svg

Yeah I know, that part from load balancer to container IP looks scary, but that’s what we are going to demystify.

# What Happens During Kubernetes Service Creation?

So for making our deployment accessible over the internet we exposed our application, by creating a service resource of type Load Balancer in K8s using the below configuration.

```yaml
kind: Service
apiVersion: v1
metadata:
  name:  nginx
spec:
  selector:
    app:  nginx
  type:  LoadBalancer
  ports:
  - name:  http
    port:  8080
    targetPort:  80
```

Run`kubectl apply -f service.yaml` to create the service type load balancer.

!https://raw.githubusercontent.com/sharadregoti/try-out/master/01-kubernetes-the-hard-way/journey-of-deployment-resource-generalize-service.drawio.svg

As you know from the previous article any K8s resource that is created gets persisted in the ETCD store via the `api-server` component. The `api-server` component notifies other system components who are responsible for handling service resource.

The other system component who get’s the notified are:

- Kube Controller Manager
- Cloud Controller Manager
- Kube Proxy

The above components decides the fate of our HTTP request. To simplify our technical journey ahead, let’s phase out our journey in sections. I have represented the journey of our HTTP request in the below diagram, we have source, destination & stop overs.

!https://raw.githubusercontent.com/sharadregoti/try-out/master/01-kubernetes-the-hard-way/journey-of-http-request-stop-overs.drawio.svg

We will start by,

1. Understanding the stop overs (highlighted in red)
2. Then, understand how the stop overs connect with each other
3. Finally, we will run through an HTTP request example

## 1. Understanding The Stop Overs

All the stop overs that you saw in the above image is created when a service type load balancer  is created.

### So let’s understand, what does the service type load balancer does?

1. When we create a service type load balancer on any Kubernetes cluster running in cloud, a load balancer is provisioned by the cloud provider. As represented by `stop-over-1`.
    
    As we know when load balancer is created a domain name is also provided by the cloud provider. That domain can be used to access our application.
    
2. In K8s the Load Balancer service type is a super set of NodePort & ClusterIP service, what that means is that the load balancer service contains both the features of NodePort & ClusterIP services.
    
    What are the features of this services you might ask:
    
    - The Node Port service opens up some ports on the worker node for external traffic to enter the cluster, represented by `stop-over-2` in the above diagram.
    - The ClusterIP service provides a static IP address which can be used inside K8s for communication, represented by `stop-over-3` in the above diagram.

### Let’s understand, the creation of stop-over-2 & stop-over-3

When `kube controller manager` is notified about a service resource & as we know from our previous article that the KCM comprises of many controllers and one of those controllers is a service controller, which takes appropriate actions on the service resource.

So here’s what the service controller does for service type load balancer:

1. First, it gets the list of pods that matches the selector field specified in the service resource (for e.g where `app: nginx`), after that it iterates over the pod list and notes down the IP address of the pod.
2. Then it creates an internal static IP address called the `clusterIP`.
3. Now it create a new K8s resource called **Endpoint** which basically maps a single IP to multiple IPs. The single IP here is the clusterIP & the multiple IPs correspond to the pod IPs.
    
    Whenever pods gets created/destroyed in K8s, it is responsibility of service controller to update the corresponding pod IP addresses in the endpoint resource. So that the cluster IP always resolves to the latest Pod IP address.
    
4. Finally, It exposes a random port (ranging from 30000-32768) on the worker node.

That’s it, this’s what the service controller does for load balancer service type.

### Let’s understand, the creation of stop-over-1

Now the question is who provisions the load balancer on the cloud? when service type load balancer is created. Surely, It is not the service controller.

Think about it, To provision a load balancer you need to interact with the cloud providers API & no Kubernetes component directly talks to the cloud provider right?

This is where the `cloud controller manager` comes into the picture. CCM is pluggable component in the K8s architecture, used to perform deep-integration with the cloud provider.

A thing note about CCM is that every cloud provider has their own implementation of CCM, so when you provision a managed K8s on cloud. The provider runs their own implementation of CCM.

The `api-server` notifies the CCM about the load balancer service, then CCM kicks into action & starts provisioning a load balancer by using cloud providers API.

**Difference between KCM & CCM**

What we learned about KCM in our previous article also applies to CCM, the operational principles around control loop apply, as does the leader election functionality.

CCM is also a collection of control loops, the difference is simply the concerns the controller address. 

KCM contains control loops that concern themselves with core K8s functionality.

The CCM will not be found in every cluster, these controller are concerned with the integration with the cloud provider, you may not have this component running in your local cluster, the CCM concerns with reconciling existing state with the cloud providers infrastructure to fulfill the desired state, they integrate with cloud provider API & often address vendor specific concerns, 

Such as provisioning **Load Balancers, Persistent Storage, VMs etc…**

## How Stop Overs Connect With Each Other?

Ok, Now we know all the stop overs. Let’s understand how the information flows through them.

!https://raw.githubusercontent.com/sharadregoti/try-out/master/01-kubernetes-the-hard-way/journey-of-http-request-final.drawio.svg

The path from LB (3) to Worker nodes(4) is configured by CCM while provisioning the LB.

For configuration it requires 2 things:

- IP address of the worker node.
- Port on which worker nodes is listening for requests.

Every worker node has it’s own IP address, CCM has access to the IP address as it can interact with the cloud providers API. The port of worker node is obtained by reading the service object, whihc is updated by KCM to reflect the random NodePort assignment.

With this, the load balancer is configured in such a way that it forwards the request it receives to any one of the configured worker node.

Data flow from Load Balancer To Node IP is easy-peasy, Totally handled by the cloud provider.

Once the request reaches the worker node, It is the responsibility of the OS to send the request to the appropriate container. Configuration of request routing on the OS level is done by configuring the IP tables of the OS.

And this configuration of IP tables is done by the `kube-proxy`component of K8s, residing on every worker node.

The thing is for the load balancer service type after KCM has done his job of assigning node port, creating clusterIP & mapping cluster IP with Pod IPs in endpoint resource.

Kube-proxy takes all this configuration & configures the IP tables accordingly. Such that any traffic received on worker node port get redirected to the container port. 

And here the journey of our HTTP request comes to an end.

---

That’s it in this blog post, If you liked this blog post. You can follow me on twitter @SharadRegoti where I talk about **Microservices, Cloud, Kubernetes & Golang.**

In the next post, We’ll be creating our own Kubernetes!!!