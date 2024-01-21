+++
title = 'Problems Faced While Running Gitlab Runners On K8s With AWS Spot Instances'
date = 2024-01-21T10:39:08+05:30
draft = false
+++

In this post, I'll share the challenges encountered and the solutions implemented while operating self-hosted Gitlab runners on Kubernetes.

Please note: this post does not cover setting up Gitlab runners—it assumes you've already done so.

The following outlines the specifics of our development EKS cluster where the Gitlab runners operate:

- Two managed node groups:
    1. **Application Node Group:** Used for running applications.
        - Instance types selected: m5.xlarge, t3.xlarge
        - Auto-scaler configuration: Min 6, Max 9
    2. **Runner Node Group:** Used for running Gitlab runners.
        - Instance types selected: t3a.2xlarge
        - Auto-scaler configuration: Min 3, Max 5
- All nodes utilize AWS spot instances.
- The EKS spans across two AZs, with each AZ containing a subnet with a CIDR range of /25 (equivalent to 256 IPs).
- A KubePromStack Helm chart is installed for cluster monitoring.

Now, let's delve into the issues we faced.

# 1. Error: Job Failed, Pod Not Found

Upon transitioning from on-demand instances to spot instances, we began encountering frequent **`runner system failure`** errors. An investigation into these errors (which involved checking kubectl events) revealed that they occurred whenever our Kubernetes pods were terminated unexpectedly.

![pod-not-found-top-error.png](https://s3-us-west-2.amazonaws.com/secure.notion-static.com/0b617a53-71d7-47c8-919c-73e71126f282/pod-not-found-top-error.png)

![pod-not-found-bottom-error.png](https://s3-us-west-2.amazonaws.com/secure.notion-static.com/1a75a8f6-6ca1-4dba-9c94-b051b805c4e9/pod-not-found-bottom-error.png)

The pods were being terminated because the AWS spot instances were reclaimed, resulting in the termination of all pods scheduled on that node.

To resolve this issue, we implemented an auto-retry mechanism in all our pipelines, as illustrated below. For more information on Gitlab retries, please refer to the official [documentation](https://docs.gitlab.com/ee/ci/yaml/#retry).

```yaml
default:
  retry:
    max: 2
    when:
      - "unknown_failure"
      - "runner_system_failure"
      - "stuck_or_timeout_failure"
      - "scheduler_failure"
      - "job_execution_timeout"  
```

# ****2. Job Failed: Execution Took Longer Than 1h0m0s Seconds****

In Kubernetes, you can specify resource (CPU & RAM) requests and limits for each pod. In the same vein, during the installation of our Gitlab runners, we configured them with the following resource requests and limits:

```yaml
cpu_request = "500m"
cpu_limit = "1000m"
memory_request = "1000Mi"
memory_limit = "2000Mi"
```

However, this configuration wasn't effective for certain jobs. We started experiencing issues where jobs that typically took 10 minutes were taking 30-40 minutes to complete, or sometimes they would timeout with this error: **`ERROR: Job failed: execution took longer than 1h0m0s seconds`**.

Upon investigation, we discovered that specific jobs such as SonarQube, unit testing, and certain Java builds consumed significantly more CPU than the specified request limit. As the CPU limit was being enforced on the pod, the process CPU was throttled, leading to a slowdown in jobs.

To resolve this, we removed **`cpu_limit`** and **`memory_limit`** from our runner configuration. While this decreased the error frequency, it didn't completely eliminate the issue.

Another CPU starvation scenario occurred when two CPU-intensive jobs were scheduled on the same node, resulting in one process starving the others.

To address this, we took the following two steps:

1. **Added Timeouts**
We observed that all of our jobs completed within 10 minutes, and those with resource allocation issues took a significant amount of time. As a result, we added a default timeout of 15 minutes to all jobs in our pipeline. 

When a job timed out, it would be automatically retried by Gitlab, as we included `**job_execution_timeout**` in the 'when' condition of the retry block.

You can learn more about adding timeouts in Gitlab from the official [documentation](https://docs.gitlab.com/ee/ci/runners/configure_runners.html#set-maximum-job-timeout-for-a-runner).
2. **Added Resource Requests, Specific to Jobs**
We created a list of jobs and matched each with the amount of resources required for proper execution.
    
    ![job-specific-resources.png](https://s3-us-west-2.amazonaws.com/secure.notion-static.com/8f83690b-fdaa-4ce4-914a-7db7bd38c133/job-specific-resources.png)
    
    Armed with this data, we adjusted our default values in the Gitlab runners with the following configuration:
    
    ```yaml
    cpu_request = "200m"
    cpu_request_overwrite_max_allowed = "6000m"
    memory_request = "200Mi"
    memory_request_overwrite_max_allowed = "8000Mi"
    ```
    
    For resource-intensive jobs, we added specific resource requests:
    
    ```yaml
    unit-testing:
      variables:
        KUBERNETES_CPU_REQUEST: "4000m"
        KUBERNETES_MEMORY_REQUEST: "4000Mi"
    ```
    
    Consequently, jobs without specified resources use the default values of 200MB and 200 milli CPU shares, while jobs with specified values run with those allocations from the start.
    

By implementing custom timeouts and job-specific resource requests, we efficiently managed CPU allocation and substantially minimized job execution time. These strategies provided a more balanced resource distribution, ensuring that jobs completed within an acceptable timeframe, thus alleviating the problems caused by CPU throttling and starvation.

# 3. Helm Upgrade Gets Stuck

The last job in our CD stage is **`deploy`**, in which we execute the helm upgrade command to roll out our applications. If AWS reclaims the spot instance during this job, we encounter the usual **`pod not found`** error. The auto retry mechanism that we implemented then attempts to rerun the job. However, the helm upgrade job continues to fail, as shown in the following error:

![helm-upgrade-another-upgrade-in-progress-error.png](https://s3-us-west-2.amazonaws.com/secure.notion-static.com/a2412dfb-7d06-4a3b-af17-6a2152ae5e9e/helm-upgrade-another-upgrade-in-progress-error.png)

This error primarily occurs due to two reasons:

1. Another upgrade is being performed on the same release elsewhere.
2. The helm CLI was unable to exit cleanly during the last helm upgrade, resulting in a **`pending-upgrade`** status, which causes all future upgrades to fail.

In our case, helm was stuck in a bad state due to the abrupt termination of pods when AWS reclaimed our spot instances.

Whenever this error occurred, we had to manually perform a **`helm rollback`** and re-run the deploy job.

Unfortunately, this is a known issue with helm and, as of now, there's no direct fix. You have to work it out on your own.

Since this is a development cluster and the production environment still has some time, we wrote a script to check the status of the helm release. If the release isn't in the **`deployed`** state, it performs a rollback automatically and then proceeds with the upgrade. However, we plan to transition our deployments to ArgoCD in the near future.

```yaml
script:
    - helm_output=$(helm status ${PROJECT} -n ${HELM_NAMESPACE} || echo "fail")
    - status=$((echo "$helm_output" | grep STATUS | awk '{print $2}') || echo "fail" ) 
    - >
      if [[ ${helm_output} != "fail" && ${status} != "deployed" ]]; then
        helm rollback ${PROJECT} -n ${HELM_NAMESPACE}
      fi
		- |
      helm upgrade -i ${PROJECT} ${CHART_PATH} \
        --values envs/_base/values.yaml \
        --values envs/${ENV}/03-service-configs/_base/values.yaml \
        --values envs/${ENV}/03-service-configs/${PROJECT}/values.yaml -n ${HELM_NAMESPACE} --atomic --timeout 1000s
```

# 4. Job Failed: Waiting for Pod Running, Timed out

As the number of nodes in our cluster grew, we began encountering this error:
`ERROR: Job failed (system failure): prepare environment: waiting for pod running: timed out waiting for pod to start. Check https://docs.gitlab.com/runner/shells/index.html#shell-profile-loading for more information`

This error typically indicates that a pod was scheduled on a node, but the kubelet was unable to start the containers specified in the pod due to various reasons, such as insufficient CPU resources, issues with the Container Network Interface (CNI), or other operations required for starting the container.

By default, the GitLab runner waits for 3 minutes (configured by the poll_timeout) for the pod to start running. If the pod's status doesn't change to **`Running`** within this time, the pod is terminated, and the aforementioned error is displayed.

![system-failure-waiting-for-pod-running-timed-out.png](https://s3-us-west-2.amazonaws.com/secure.notion-static.com/586b59df-ea60-413a-974d-59900f1d03cc/system-failure-waiting-for-pod-running-timed-out.png)

In our case, when this error occurred, our pod was stuck in the **`init`** state, and describing the pod showed an event **`vpc-cni failed to assign an IP address`**.

![pod-stuck-init-cpu.png](https://s3-us-west-2.amazonaws.com/secure.notion-static.com/7f349efe-f68b-4470-a4f0-b4f557a8f17a/pod-stuck-init-cpu.png)

Since we are using Amazon EKS with vpc-cni, the issue arose when the vpc-cni couldn't allocate an IP address to the pod as the subnet in which the node was running ran out of IP addresses. This was due to inadequate VPC planning and a lack of understanding about how vpc-cni works in default mode.

### **VPC CNI Working In Default Mode**

In default mode, the vpc-cni assigns an IP address to a pod from the subnet where the node is running, utilizing the Elastic Network Interface (ENI) attached to the node. The number of ENIs that can be attached and the number of IPs available per ENI depend on the instance type you've selected. For a **`t3a.2xlarge`** node, it can attach 4 ENIs each having 15 IP addresses.

When a node starts up, it takes all IPs available in a single ENI (15 in our case), even when there is no pod scheduled on it. However, the **`WARM_ENI_TARGET`** variable of vpc-cni is set to 1 by default. Thus, when a single pod gets scheduled on a node, vpc-cni ensures there is always an additional ENI worth of IP addresses available and attaches another ENI (another 15 IPs, totalling 30).

When subnet is near depletion of IPs and a pod gets scheduled on a node, vpc-cni tries to obtain an IP from the subnet. If there are no IPs available, the container fails to start. GitLab waits until the poll_timeout and then terminates the pod, resulting in this error.

### **Our Solution**

Instead of warming an entire ENI at once, we decided to control how many IPs get initialized on node startup and how many IPs are kept in a warm state using the following variables:

1. **MINIMUM_IP_TARGET:** How many IPs are initialized on node startup?
2. **WARM_IP_TARGET:** How many IPs are kept in a warm state?

```yaml
- name: MINIMUM_IP_TARGET
  value: "15"
- name: WARM_IP_TARGET
  value: "5"
```

By setting these variables, we can better manage our subnet IP usage and prevent the error of **`vpc-cni failing to assign an IP address`**, which ultimately helps avoid the job failing due to pods timing out during startup.

# 5. Pod Ephemeral Storage Problems

As our services expanded, we needed to run more jobs concurrently, prompting us to increase our runner concurrency from 7 to 15. However, this adjustment led to the emergence of errors like **`no space left on device`** and **`Job Failed: pod status is failed`**.

![storage-problm-status-is-failed.png](https://s3-us-west-2.amazonaws.com/secure.notion-static.com/424bd86f-ec96-409f-a6b7-4e3620f48465/storage-problm-status-is-failed.png)

![storage-problem-no-space-left.png](https://s3-us-west-2.amazonaws.com/secure.notion-static.com/bf10ba0e-9e34-4c39-8dad-da0061225829/storage-problem-no-space-left.png)

![storage-problem-git-fails.png](https://s3-us-west-2.amazonaws.com/secure.notion-static.com/8d50d6de-1380-4898-b69c-e13969ac790f/storage-problem-git-fails.png)

But the root cause of the other error, **`Job Failed: pod status is failed`**, was not immediately evident from the GitLab logs. Following an investigation and review of the kubectl events, we determined that pods were being evicted due to storage issues, with the error message stating: **`The node was low on resource: ephemeral-storage. Container build was using 72Ki, which exceeds its request of 0. Container helper was using 1464696Ki, which exceeds its request of 0`**.

While all our nodes were operating with 20GB of disk storage, we overlooked the storage requirements of GitLab jobs, which involve artifacts, git repositories, and more. With the increased concurrency, more pods were scheduled on the same node, leading to a heightened storage demand.

The solution was straightforward. By increasing node storage from 20GB to 50GB, we effectively addressed the issue, preventing further pod evictions due to insufficient ephemeral storage.

---

That’s it in this blog post, If you liked this blog post. You can also checkout my YouTube channel where we talk about **Microservices, Cloud & Kubernetes,** you can check it out here 👉[link](https://www.youtube.com/@techwithsharad)