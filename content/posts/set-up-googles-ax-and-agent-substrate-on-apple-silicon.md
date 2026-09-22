---
title: "How to Set Up Google's AX and Agent Substrate Locally on Apple Silicon"
date: 2026-09-22T21:00:00+05:30
draft: false
description: "A working setup guide for Google's AX agent orchestrator and Agent Substrate on a kind cluster, including five real bugs I hit on Apple Silicon and how to get past each one."
---


[AX](https://github.com/google/ax) is Google's new take on running agent workloads on Kubernetes: you declare a `Task`, a `Workspace`, a `Gateway`, and a `Model` as YAML, and it sandboxes the whole thing, wires up the workspace, fences the network, and lets you suspend and resume the agent like a checkpointed VM. It sits on top of [Agent Substrate](https://github.com/agent-substrate/substrate), Google's sandboxed execution runtime, which is the part that actually does the suspend/resume and the gVisor isolation.

Both repos are brand new and say so upfront ("we are still actively refining our core concepts"). I found that out the hard way. The README's three-step quickstart doesn't get you to a working setup on its own, at least not on a Mac. I hit five separate bugs getting from "clone the repo" to "suspend and resume a task," and four of them turned out to already be filed on GitHub, which was oddly reassuring.

This is the guide I wish I'd had: the actual commands that work, and the exact point where the documented path breaks.

## What you need

- Go 1.27+
- Docker Desktop, running
- `kubectl`
- An Apple Silicon Mac (this guide calls out the arm64-specific fix; skip it on Intel/amd64)

`kind` gets installed automatically as a Go tool by Agent Substrate's own scripts, so you don't need it on your `PATH` beforehand.

## Step 1: Install the ax CLI

```bash
go install github.com/google/ax/cmd/ax@latest
```

This lands in `$(go env GOPATH)/bin`. Make sure that's on your `PATH` before moving on.

## Step 2: Clone both repos

```bash
git clone https://github.com/google/ax.git
git clone https://github.com/agent-substrate/substrate.git
```

You need both. AX's control plane (`ax-controller`, `ax-server`, Redis) is what `ax apply` talks to, but it does nothing on its own. It hands actual sandboxing off to Agent Substrate, which needs its own control plane (`ate-api-server`, `ate-controller`, atelet, plus Postgres and an S3-compatible store called rustfs) running in the same cluster.

## Step 3: Create a kind cluster with a local registry

Skip Docker Desktop's built-in Kubernetes for this. Agent Substrate's own dev quickstart is built around a `kind` cluster with a local image registry, and going along with that instead of fighting it saved me a lot of time.

```bash
cd substrate
hack/create-kind-cluster.sh
```

This creates a `kind` cluster, a registry container on `localhost:5001`, labels the node with `ate.dev/substrate-version` (workers only schedule on labeled nodes), and probes for `/dev/kvm` to decide whether microVM support gets enabled. On Docker Desktop for Mac, `/dev/kvm` isn't there, so you get gVisor-only sandboxing. That's fine for this guide, just worth knowing before you go looking for the microVM tier and can't find it.

## Step 4: Deploy Agent Substrate

```bash
hack/install-ate-kind.sh --deploy-ate-system
```

This builds and deploys the whole substrate control plane with `ko`, plus Postgres and rustfs, and waits for every rollout to finish. It pulls something like 570MB of third-party images the first time, so give it a few minutes. Once it's done, `kubectl get pods -n ate-system` should show everything Running, including a `postgres-0` and a `rustfs-*` pod.

## Step 5: Deploy AX's control plane

```bash
cd ../ax
make deploy AX_IMAGE_REPO=localhost:5001/ax
```

This deploys Redis, then builds and pushes `ax-controller` and `ax-server` through `ko` to your local registry. Check it:

```bash
kubectl get pods -n ax-system
```

`ax-redis` and `ax-server` come up clean. `ax-controller` will crash-loop right now, and that's expected. Its logs say why:

```
reading CA file "/run/servicedns-ca/trust-bundle.pem": open /run/servicedns-ca/trust-bundle.pem: no such file or directory
```

That file comes from a `ClusterTrustBundle` that Agent Substrate provisions. If you did Step 4 before this, give it another minute and it'll resolve on its own once the controller retries. If it's still crash-looping after that, double check `ate-system` actually finished rolling out.

## Step 6: Build a task-runner image that actually works

The example manifest ships with this image reference:

```
gcr.io/ax-substrate/ate-images/ax-task-runner@sha256:3a0dea6a...
```

That's a private Google registry. You'll get `DENIED: Unauthenticated request` trying to pull it, so you need to build your own from `Dockerfile.task-runner`. This is where I hit two bugs back to back.

First: the repo's own `.dockerignore` excludes `bin/`, which is exactly where the Makefile puts the compiled binary the Dockerfile then tries to `COPY`. Following the documented `make build-task-runner` fails every time with something like:

```
failed to calculate checksum of ref ...: "/bin/linux_amd64/ax-task-runner": not found
```

Already filed: [google/ax#364](https://github.com/google/ax/issues/364). The workaround is to build the binary, then hand `docker build` a clean context that doesn't have the repo's `.dockerignore` in it at all:

```bash
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" \
  -o /tmp/task-runner-ctx/bin/linux_amd64/ax-task-runner ./cmd/ax-task-runner

mkdir -p /tmp/task-runner-ctx/cmd/ax-task-runner
cp cmd/ax-task-runner/antigravity_bootstrap.py /tmp/task-runner-ctx/cmd/ax-task-runner/

docker build --platform linux/arm64 -t localhost:5001/ax/ax-task-runner:latest \
  -f Dockerfile.task-runner /tmp/task-runner-ctx

docker push localhost:5001/ax/ax-task-runner:latest
```

Second, and the one that actually cost me time: notice `GOARCH=arm64` up there. The Makefile hardcodes `amd64`. Build amd64 like the docs say on an Apple Silicon Mac, and the image pulls fine, then fails with an error buried three log lines deep inside the gVisor sandbox's own output, nowhere `ax describe task` shows you:

```
failed to load /usr/local/bin/ax-task-runner: exec format error
```

Regular pods (`ax-controller`, `ax-server`) run fine as amd64 on an arm64 node because Docker Desktop emulates them transparently. gVisor's `runsc` doesn't go through that path. It executes the binary directly inside its own sandboxed kernel, so a wrong-arch binary just fails to exec. Build for `arm64` and it works. Already filed: [google/ax#358](https://github.com/google/ax/issues/358).

Grab the digest from the `docker push` output and update `examples/task.yaml`'s `spec.image` to point at your own `localhost:5001/ax/ax-task-runner@sha256:...`.

## Step 7: Create a WorkerPool

Apply the example task now and it fails immediately:

```
ActorResumeFailed: resuming actor default/task123: rpc error: code = ResourceExhausted
desc = no free workers available
```

Nothing in AX's docs mentions this, but a task can't run anywhere until a `WorkerPool` exists, and a fresh install has zero. Already filed: [google/ax#367](https://github.com/google/ax/issues/367).

Grab your node's substrate label first:

```bash
kubectl get nodes -o jsonpath='{.items[0].metadata.labels.ate\.dev/substrate-version}'
```

Drop that value into a `WorkerPool` manifest:

```yaml
apiVersion: ate.dev/v1alpha1
kind: WorkerPool
metadata:
  name: ax-default
  namespace: ax-system
spec:
  replicas: 2
  workerImage: ko://github.com/agent-substrate/substrate/cmd/ateom-gvisor
  template:
    nodeSelector:
      ate.dev/substrate-version: "PASTE_THE_VALUE_HERE"
    resources:
      limits:
        cpu: "1"
        memory: 1Gi
      requests:
        cpu: 250m
        memory: 1Gi
```

Then apply it from the `substrate` repo, since `ko` needs to resolve `ateom-gvisor` against that module:

```bash
cd ../substrate
KO_DOCKER_REPO=localhost:5001/substrate ko apply -f workerpool.yaml
```

The `ActorTemplate` AX generates per task has no `workerSelector` at all ([google/ax#368](https://github.com/google/ax/issues/368)), so any pool in the cluster will pick it up. That's fine for a single-pool local setup; it stops being fine the moment you have more than one.

## Step 8: Point AX at a snapshot bucket that exists

The task should run now. Suspend/resume, AX's actual headline feature, still won't:

```
FAILED_SAVE_SNAPSHOT: ... NoSuchBucket: The specified bucket does not exist
```

`ax-controller` defaults its snapshot storage to `gs://snapshot-substrate-test-ax-substrate/...`, a Google-internal bucket. Agent Substrate's own kind install provisions a *different* bucket, `ate-snapshots`, in the local rustfs store. Already filed: [google/ax#372](https://github.com/google/ax/issues/372). Point the controller at the one that actually exists:

```bash
kubectl set env deployment/ax-controller -n ax-system \
  AX_SNAPSHOTS_BUCKET=gs://ate-snapshots/ate-env/
```

## Step 9: Run it

```bash
ax apply -f examples/task.yaml
ax get tasks
ax describe task task123
```

You should see `Phase: Running` with `GatewayReady`, `WorkspaceReady`, and `Ready` all green.

## The part that's actually worth the setup pain

```bash
ax suspend task task123
# ...
ax resume task task123
```

On my machine, `SuspendActor` took about 700ms and `ResumeActor` about 350ms. More interesting than the speed: the worker IP changed from one suspend/resume cycle to the next. The actor came back on a completely different worker pod, with its `/workspace` git clone intact and correctly timestamped from before the suspend, not freshly re-cloned.

That's the actual pitch for this over a plain Kubernetes Job: state that survives a move to different physical hardware, sub-second, and you can `ax ssh <task> -- <command>` in and poke around while it's live.

## What's still broken

`ax ssh <task>` with no trailing command is documented as opening an interactive shell. It doesn't. It returns instantly with no output. The client defaults to running `/bin/sh`, but the underlying `Exec` call never wires up stdin, so the shell hits EOF immediately and exits, same as `echo | sh` locally. One-off commands (`ax ssh <task> -- ls -la /workspace`) work fine, since they don't need stdin. I filed this one: [google/ax#374](https://github.com/google/ax/issues/374).

Also worth knowing: deleting an actor whose ActorTemplate points at a missing bucket fails the same way suspend does, and gets stuck in `Terminating` permanently, since Substrate tries to list objects in that bucket to clean them up and 404s instead. If you hit Step 8's bug before fixing it, you may need to create the old bucket name too, just so the stuck actor can delete itself, then delete and reapply.

## Is this a Kubernetes alternative for agents?

Sort of, and it's honest about not being finished yet. The primitives are the right shape: sandboxed execution, network fencing by default, a workspace abstraction that pre-wires git and MCP servers, and suspend/resume that genuinely moves state across workers instead of just leaving a container running. None of that exists in plain Kubernetes for this kind of workload.

What it isn't yet is something you'd hand to someone who can't read Go source to debug a `runsc start: exit status 128` error. Every bug above needed either the controller logs, the atelet logs, or the actual reconciler code to diagnose, because the surfaced error rarely matched the root cause. Given both repos are pre-1.0 and say so, that's a fair trade for now. If you're evaluating this for real use, budget time for exactly this kind of digging, and check the [issues list](https://github.com/google/ax/issues) before you go looking for root causes yourself.
