# Kubernetes Services: Cluster IP, NodePort, Loadbalancer &&nbsp; Ingress, Ingress Controllers

# Introduction

Kubernetes networking principles are really confusing 😖, especially if you're a beginner. 

Regardless of how well I follow the official documentation. I honestly didn’t understand a single damn thing !!!

After watching some tutorials on Youtbube I was able to write & duplicate the tutorials, but core understanding of Kubernetes networking concepts was still missing, as sometimes I myself used to get confused about what to use where...

This blog post is my take to explain kubernetes networking concepts in depth & answer the below questions that I had that always confused me...

- How Kubernetes networking resource solves **Service Discovery** problem in Kubernetes?
- Does a Load Balancer Service really provisions a Load Balancer automatically? What will happen if I create this service locally?
- How does a production ready Kubernetes cluster expose it’s applications?
- What is the difference between Ingress & Ingress Controller?
- and more...

## Let’s Get Started

Throughout the entire blog post, we will assume the following. 

- You have familiarity with the structure of Kubernetes YAML resource definition
- You have deployed an imaginary Kubernetes cluster on the cloud across 3 VMs, the VMs/nodes have the following **Public IP addresses**
    - Node A (192.168.0.1)
    - Node B (192.168.0.2)
    - Node C (192.168.0.3)
- A microservices application having 4 services is deployed on that K8s cluster. the application is deployed in such a way that each VM has at least 1 replica of that service running
    - Products
    - Reviews
    - Details
    - Ratings
- Notations
    - $Reviews_A$ denotes that the `Reviews` service is running on node A
    - **K8s** denotes Kubernetes
    - **LBs** denotes Load Balancers

This sentence uses `$` delimiters to show math inline:  $\sqrt{3x-1}+(1+x)^2$


![Kubernetes Cluster](/images/2022-06-24-01-explained-kubernetes-services-ingress/blog-architecture.drawio_(2).svg)

Kubernetes Cluster

Let's start with answering the **why**

# 🤔 Why do we need services in the first place?

In the K8s cluster, our application code is running inside containers using the K8s **Deployment** resource. This deployment resource internally creates a K8s **Pod** resource which in turn actually runs the containers.

K8s assigns an IP address whenever a pod is created. The command below shows you an example IP address of the Pod

```yaml
kubectl get pods <pod-name> -o wide
```

![Example Pod IP](/images/2022-06-24-01-explained-kubernetes-services-ingress/pod-ip.png)

Example Pod IP

In our application, $Products_A$ Pod can use the IP address assigned to $Reviews_A$ Pod for communication. Well this technique works..., but it's not an elegant solution the reason being **K8s pods are short-lived/ephemeral.**

So if a pod dies for any reason, K8s will automatically restart the Pod, but the IP address assigned to that pod also changes. This in turn will lead to communication failure between the $Products_A$ & $Reviews_A$ micro-service as the IP address of $Reviews_A$ service is hardcoded in the configuration file of $Products_A$ service. This is problematic !!!

Wouldn’t it be great if somehow a static IP address or a DNS name gets assigned to the Pod? This will solve our above problem.

That’s where the **Service** resource of K8s comes to the rescue. I know the name Service sounds deceiving. A Service ****resource doesn't run our containers. **It merely provides and maintains a network identity for our Pod resource.**

This **Service** network identity provides both a static IP address & a DNS name, we just need to specify the ports on which the incoming request will be received & the port of the pod on which the request has to be sent.

Services listen on a static address that doesn't change & forwards the request to the unreliable container address. The way it works is that internally it keeps a map that always contains the latest IP address of the Pod. so whenever a Pod restarts it updates that map automatically so you get a seamless connection.

Another benefit of using services is load balancing between replicas of the same application, With our imaginary application as there are 3 instances of each micro-service running on separate VMs, A service will load balance the request between the replicas of  $Products_A$ $Proudcts_B$ $Products_C$.

![Load Balancing between Pods in K8s using Service Resource](/images/2022-06-24-01-explained-kubernetes-services-ingress/kubernetes-service-load-balancing.drawio.svg)

Load Balancing between Pods in K8s using Service Resource

Till now we have understood what are services & why are they needed, Let's check out the different services that K8s offers with their use cases

# 🤔 How do I make $Products_A$ service talk to $Reviews_A$ service?

### ClusterIP Service to the rescue

The following YAML defines a service of type ClusterIP

```yaml
apiVersion: v1
kind: Service
metadata:
  labels:
    app: reviews
  name: reviews
spec:
  ports:
  - name: http
    port: 5000
    protocol: TCP
    targetPort: 80
  selector:
    app: reviews
  type: ClusterIP
```

The important fields in these YAML are

`targetPort`: denotes the container port on which the request has to be forwarded

`port`: denotes the port on which the service accepts incoming requests

`selector`: This field is used for associating/attaching a K8s **Service** resource to a K8s **Pod** resource, this is done by matching labels (key-value pairs) of the pod with selectors labels specified in the selector section.

<aside>
📌 Selectors labels defined in the service performs **AND** operation for attaching a service to a pod, meaning all the labels must match for attachment to be done

</aside>

The command below shows you the static IP address of the service which can be used by other Pods

```bash
kubectl get service <service-name>
```

![clusterip-kubectl-service.png](/images/2022-06-24-01-explained-kubernetes-services-ingress/clusterip-kubectl-service.png)

You can use the above IP address for communication between $Products_A$ & $Reviews_A$ micro-service 

To communicate using DNS, there are 2 ways

- Pods residing in the same K8s **Namespace** can use the name of the service for DNS resolution'
    - Example → `http://reviews:5000`
- Pods residing in other namespaces can use the following DNS notation → `<service-name>.<namespace>.svc.cluster.local`
    - Let's say $Products$ Pod resides in **products namespaces** & $Reviews$ pod resides in **reviews namespace**, then for $Products$ pod to communicate with $Reviews$ pod you would use the following DNS → `http://reviews.reviews.svc.cluster.local:5000`

Now you can pass these addresses to your microservice for communication.

![Internal Communication Using Cluster IP Service](/images/2022-06-24-01-explained-kubernetes-services-ingress/kubernetes-cluster-ip.svg)

Internal Communication Using Cluster IP Service

Till now we understood how applications inside the K8s cluster talk to each other, but you made your application to be consumed by users on the internet. Let’s check out how it is done.

# 🤔 How do I expose $Products_A$ service over the internet?

Every application that is deployed in K8s by default cannot be accessed over the network/internet, they need to be exposed Via a **Services**. K8s provides 2 **Services** to do just that

### NodePort Service

The following YAML can be used to create a service of type NodePort

```yaml
apiVersion: v1
kind: Service
metadata:
  labels:
    app: reviews
  name: reviews
spec:
  ports:
  - name: http
    port: 5000
    protocol: TCP
    targetPort: 80
  selector:
    app: reviews
  type: NodePort
```

The above YAML definition is very similar to the **ClusterIP Service** definition, the only field that has changed is the `type` field from **ClusterIP to NodePort**. When you create a NodePort service, K8s will randomly select any available port on the node/VM between **30000 - 32767** and listen on it for incoming requests.

<aside>
📌 You can specify the nodeport field yourself in the service definition, but you have to deal with hassle checking if the port is available on every node

</aside>

After creating the service, To get the port on which this NodePort service is listening for requests use the below command

```bash
kubectl get service <service-name>
```

![node-port-service.png](/images/2022-06-24-01-explained-kubernetes-services-ingress/node-port-service.png)

In our imaginary K8s cluster, this will happen on all the nodes

- Node A (192.168.0.1:30519)
- Node B (192.168.0.2:30519)
- Node C (192.168.0.3:30519)

You can access your application using the address **nodeIP:30519**, for Node A it would be 192.168.0.1:30519.

![Exposing Application Using Nodeport Service](/images/2022-06-24-01-explained-kubernetes-services-ingress/kubernetes-nodeport-service.svg)

Exposing Application Using Nodeport Service

But this is not practical, users of your application won’t remember your IP address & port for accessing your app. They need a domain name like [`myapp.com`](http://myapp.com) to access your app. We solve this by putting an external load balancer in front of our VMs. The load balancers public IP is mapped to myapp.com. So when you type myapp.com on the browser it resolves to load balancer Ip, Then the load balancer forwards these requests to any of our 3 nodes according algorithm configured in load balancer.

K8s provides another service to expose your application, it is called the Load Balancer service.

### Load Balancer

Following Yaml can be used a create a service of type Load Balancer

```yaml
apiVersion: v1
kind: Service
metadata:
  labels:
    app: reviews
  name: reviews
spec:
  ports:
  - name: http
    port: 5000
    protocol: TCP
    targetPort: 80
  selector:
    app: reviews
  type: LoadBalancer
```

The above YAML definition is very similar to the **ClusterIP Service** definition, the only field that has changed is the `type` field from **ClusterIP to LoadBalancer**.

Load Balancer service is an abstraction over the NodePort service, What’s special about this service is that If you are using a managed Kubernetes service like EKS(AWS), GKS(GCP), AKS(Azure) or any other cloud provider, it provisions a load balancer by itself so that you don’t have to hassle setting up a load balancer yourself.

After creating the service, To get the public IP address of the load balancer provisioned by this service use the below command

```bash
kubectl get service <service-name>
```

![Load Balancer Service Example](/images/2022-06-24-01-explained-kubernetes-services-ingress/load-balancer-ip.png)

Load Balancer Service Example

But what happens if you create a LoadBalancer service on your local K8s cluster or configured your own K8s from scratch on the cloud?

The answer is it won’t do anything, you will see a **<pending>** state for the IP address section. The reason is cloud-specific K8s clusters add other special programs/controllers which detects the Load Balancer service creation and takes the appropriate action accordingly.

One thing to note here, Loadbalancer service doesn’t expose any ports by itself like the NodePort service

**LoadBalancer** is a superset of **NodePort** is a superset of **clusterIP** service, which means when you create a service of type Nodeport a ClusterIP service also gets created implicitly. So if you create a NodePort service for $Products$ microservice, then this same service can be used for both internal communications by other pods & accessing the $Products$ service on the internet.

![Service Superset](/images/2022-06-24-01-explained-kubernetes-services-ingress/superset.svg)

Service Superset

# 🤔 What should we use NodePort or LoadBalancer service?

If you read carefully, you will observe there is not much difference between the NodePort & LoadBalancer service. Except for some automation being done in the latter one. 

Using a Load Balancer service takes away the trivial task of configuring & provisioning LBs. Whereas using a NodePort gives you the freedom to set up your own load-balancing solution, such that to configure environments that are not fully supported by Kubernetes, or even to expose one or more nodes' IPs directly.

So it depends upon the situation you are in, but the general rule of thumb is to try to use LoadBalancer service first if it doesn’t work for your environment go for NodePort service

That is it, these two services are used to expose your application outside of the cluster. But there is a slight problem. 

If you want to expose more than one application, you end up creating multiple NodePort / LoadBalancer Services. This gets the job done but you have to face some consequences because with each NodePort service you need to deal with the hassle of manually managing the IPs & ports. And with each LoadBalancer service, you go on increasing your cloud bill.

No worries 😟, K8s has addressed this issue with the use of **Ingress & Ingress controllers.**

# 🤔 How to expose multiple applications without creating multiple services

### Ingress & Ingress controller

Ingress & Ingress controller are two different things, people usually get confused with these terms

An Ingress controller is just another K8s Deployment resource (an app running in a container), but what is special about this deployment is that it provides a single point of control for all incoming traffic into the Kubernetes cluster

Think of **Ingress controller** as a smart proxy running in K8s & **Ingress** is a K8s resource that configures the routing logic of **Ingress controller.**

<aside>
💡 Analogy, For people who have worked with Nginx (Nginx the webserver will be your Ingress controller, the nginx.conf file will be your Ingress resource)

</aside>

With these, you can specify what request needs to be routed to which service in your k8s cluster on the basis of any request parameter like host, URL path, sub domain etc...

Following Yaml can be used to create Ingress resource

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: ingress-wildcard-host
spec:
	ingressClassName: nginx
  rules:
  - host: "products.myapp.com"
    http:
      paths:
      - pathType: Prefix
        path: "/products"
        backend:
          service:
            name: products-app
            port:
              number: 80
  - host: "ratings.foo.com"
    http:
      paths:
      - pathType: Prefix
        path: "/ratings"
        backend:
          service:
            name: ratings-app
            port:
              number: 80
```

With the above YAML, we have exposed two services, if a request comes from `products.myapp.com` and the request path starts with `/proudcts` we send it to our products-app **K8s Service** which in turn forwards the request to our container. 

Similarly, if a request comes from `ratings.bar.com` it is forwarded to ratings-app

### Ingress YAML Explaination

The important fields in these YAML are

`ingressClassName`: Usually there is only one ingress controller in the cluster, but no one has stopped you from creating many. So this field is used for the selection of ingress controller, similar to the selector field of services

`service`: denotes the name of service on which the request has to be forwarded

An ingress resource should be created in the same namespace where the corresponding **K8s Service** resides, But an ingress controller can be placed in any namespace, it will automatically detect Ingresess defined in other namespaces on the basis of `ingressClassName`.

An important thing to remember an Ingress by itself doesn’t expose any port. In the end, it's just a K8s deployment resource. You need a service(NodePort/LoadBalancer) in front of it to expose it.

![Exposing Application Using Load Balancer Service](/images/2022-06-24-01-explained-kubernetes-services-ingress/production-setup.drawio.svg)

Exposing Application Using Load Balancer Service

## Conclustion

Congratulation for sticking to the end, Let’s summarize what we have learned so far.
Kubernetes gives us 3 types of Service Resource Object

- **Cluster IP Service**→
    - Used for internal communication amoung workloads & solves **Service Discovery.**
- **Nodeport Service** →
    - Used for exposing applications over the internet, mostly used in development environments
- **Load Balancer Service** →
    - Used for exposing applications over the internet
    - Provision actual load balancer on the cloud, on supported cloud platforms
- **Ingress Resource** →
    - Used for controlling incoming traffic in the Kubernetes cluster
    - Various implementations are available each with it’s pros & cons

That’s it in this blog post, If you liked this blog post. You can also checkout my YouTube channel where we talk about M**icroservices, Cloud & Kubernetes,** you can check it out here 👉[link](https://l.linklyhq.com/l/KhmP)

If you have any questions you can reach me on Twitter at [@SharadRegoti](https://twitter.com/SharadRegoti)