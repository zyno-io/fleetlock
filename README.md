# fleetlock
[![GoDoc](https://pkg.go.dev/badge/github.com/poseidon/fleetlock.svg)](https://pkg.go.dev/github.com/poseidon/fleetlock)
[![Quay](https://img.shields.io/badge/container-quay-green)](https://quay.io/repository/poseidon/fleetlock)
[![Workflow](https://github.com/poseidon/fleetlock/actions/workflows/build.yaml/badge.svg)](https://github.com/poseidon/fleetlock/actions/workflows/build.yaml?query=branch%3Amain)
[![Sponsors](https://img.shields.io/github/sponsors/poseidon?logo=github)](https://github.com/sponsors/poseidon)
[![Mastodon](https://img.shields.io/badge/follow-news-6364ff?logo=mastodon)](https://fosstodon.org/@poseidon)

`fleetlock` is a reboot coordinator for Fedora CoreOS nodes in Kubernetes clusters. It implements the [FleetLock](https://coreos.github.io/zincati/development/fleetlock/protocol/) protocol for use as a [Zincati](https://github.com/coreos/zincati) lock [strategy](https://github.com/coreos/zincati/blob/master/docs/usage/updates-strategy.md) backend.

## Usage

Zincati runs on-host (`zincati.service`). Declare a Zincati `fleet_lock` strategy when provisioning Fedora CoreOS nodes. Set `base_url` for host nodes to access the in-cluster `fleetlock` Service (e.g. known ClusterIP).

```yaml
variant: fcos
version: 1.4.0
storage:
  files:
    - path: /etc/zincati/config.d/55-update-strategy.toml
      contents:
        inline: |
          [updates]
          strategy = "fleet_lock"
          [updates.fleet_lock]
          base_url = "http://10.3.0.15/"
```

Apply the `fleetlock` Deployment, Service (with ClusterIP), and ServiceAccount.

```
kubectl apply -f examples/k8s
```

Inspect the fleetlock Lease object.

```
$ kubectl get leases -n default
NAME                HOLDER                             AGE
fleetlock-default   049ad0f57ade4723a48692b7b692c318   4m50s
```

### Configuration

Configure the server via flags.

| flag       | description  | default      |
|------------|--------------|--------------|
| -address   | HTTP listen address | 0.0.0.0:8080 |
| -log-level | Logger level | info |
| -drain-max-wait | Maximum time to wait for pod evictions during drain | 60s |
| -maintenance-window-start | Start of maintenance window in HH:MM format (UTC) | (none) |
| -maintenance-window-end | End of maintenance window in HH:MM format (UTC) | (none) |
| -lock-cooldown | Minimum wait after a node reboots/unlocks before granting the next lock (0 disables) | 5m |
| -slack-bot-token | Slack Bot User OAuth Token for notifications | (none) |
| -slack-channel-id | Slack channel ID for lock/unlock notifications | (none) |
| -version   | Show version | NA   |
| -help      | Show help    | NA   |

Or via environment variables.

| variable   | description            | default   |
|------------|------------------------|-----------|
| NAMESPACE  | Kubernetes Namespace   | "default" |
| KUBECONFIG | Development Kubeconfig | NA        |
| SLACK_BOT_TOKEN | Slack Bot User OAuth Token (overrides -slack-bot-token flag) | NA |

#### Maintenance Windows

When `-maintenance-window-start` and `-maintenance-window-end` are both set, lock requests outside the window are denied with HTTP 403. Cross-midnight windows are supported (e.g. `22:00` to `06:00`). Times are in UTC.

```
./bin/fleetlock -maintenance-window-start 02:00 -maintenance-window-end 06:00
```

#### Reboot Cooldown

After a node reboots and releases its lock, fleetlock waits `-lock-cooldown` (default `5m`) before granting the next reboot lock, giving the rebooted node time to recover its pods. Requests during the cooldown are denied with HTTP 429 and a `reboot_cooldown` reply, so Zincati retries later.

The cooldown is only armed when the node *actually rebooted* — detected by its Kubernetes `Ready` condition transitioning after the lock was acquired (it went `NotReady` and recovered). A lock that completes without a reboot does not start a cooldown. Set `-lock-cooldown 0` to disable.

```
./bin/fleetlock -lock-cooldown 5m
```

#### Node Attributes

fleetlock records reboot lock state on each Node as annotations, for visibility (`kubectl get node -o jsonpath`):

| annotation | description |
|------------|-------------|
| `fleetlock.poseidon.coreos.com/state` | `locked` or `unlocked` |
| `fleetlock.poseidon.coreos.com/last-lock-time` | RFC3339 UTC time the lock was last granted |
| `fleetlock.poseidon.coreos.com/last-unlock-time` | RFC3339 UTC time the lock was last released |

The group's most recent reboot-unlock time is also stored as a `last-unlock-time` annotation on the `fleetlock-<group>` Lease, where it drives the cooldown.

#### Slack Notifications

When a Slack bot token and channel ID are configured, fleetlock posts notifications on lock grants and releases. The message includes the resolved Kubernetes node name.

```
./bin/fleetlock -slack-channel-id C0123ABCDEF
```

Set the bot token via environment variable to avoid exposing it in process arguments:

```
export SLACK_BOT_TOKEN=xoxb-...
```

### Typhoon

For Typhoon clusters, add the Zincati config a [snippet](https://typhoon.psdn.io/advanced/customization/#fedora-coreos).

```tf
module "nemo" {
  ...
  controller_snippets = [
    file("./snippets/zincati-strategy.yaml"),
  ]
  worker_snippets = [
    file("./snippets/zincati-strategy.yaml"),
  ]
}
```

## Manual Intervention

`fleetlock` coordinates OS auto-updates to avoid concurrent node updates or a potential bad auto-update continuing. Zincati obtains a reboot lease lock before finalization (i.e reboot).

If an auto-update fails, the lease continues to be held by design. An admin should investigate the node failure and decide whether it is safe to remove the lease.

```
$ kubectl get leases
$ kubectl delete lease fleetlock-default
```

## Metrics

`fleetlock` serves Prometheus `/metrics` from Go, process, and custom collectors.

| name                 | description                                         |
|----------------------|-----------------------------------------------------|
| fleetlock_lock_state | State of the fleetlock lease (0 unlocked, 1 locked) |
| fleetlock_lock_transition_count | Number of fleetlock lease transitions    |
| fleetlock_lock_request_count   | Number of lock requests   |
| fleetlock_unlock_request_count | Number of unlock requests |
| fleetlock_cooldown_denial_count | Number of lock requests denied due to the post-reboot cooldown |
| fleetlock_reboot_within_lock_count | Number of unlocks where the node rebooted during its lock window |

## Development

To develop locally, build and run the executable.

### Static Binary

Build the static binary.

```
make build
```

### Container Image

Build the container image.

```
make image
```

### Run

Run the executable.

```
export KUBECONFIG=some-dev-kubeconfig
./bin/fleetlock
```

Use curl to emulate a Zincati FleetLock client.

```json
{
  "client_params": {
    "id": "c988d2509fdf5cdcbed39037c56406fb",
    "group": "default"
  }
}
```

Request a reboot lock.


```
curl -H "fleet-lock-protocol: true" -d @examples/body.json http://127.0.0.1:8080/v1/pre-reboot
```

Release a reboot lock.

```
curl -H "fleet-lock-protocol: true" -d @examples/body.json http://127.0.0.1:8080/v1/steady-state
```

## Related

* [Zincati Guide](https://docs.fedoraproject.org/en-US/fedora-coreos/auto-updates/)
* [Zincati Docs](https://github.com/coreos/zincati/blob/master/docs/usage/updates-strategy.md)
* [FleetLock Protocol](https://coreos.github.io/zincati/development/fleetlock/protocol/)
