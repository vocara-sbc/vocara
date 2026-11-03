# contracts — Cross-Boundary Contracts

`vocara/contracts` is the interface contract module shared by the business and core sides. It defines event and HTTP data structures (DTOs) that cross service boundaries, providing a compile-time stable interface surface between the public (vocara) and private (core) repositories. Both the business and core sides build against this module without compile-time dependencies on each other's source code.

## Constraints

- **Zero third-party dependencies**: Uses only the Go standard library and struct tags, so either side can import it without additional dependencies.
- **Contracts only, no implementations**: Contains no business logic, persistence models, or transport details.
- **Versioned organization**: HTTP DTOs are organized into `domain/version` directories (such as `call/v1`), with incompatible revisions handled by a new version.

## Contents

| Path | Contents |
|---|---|
| `event/allocated.go` | `AllocatedCall`: Call allocation event with topic `queue.dequeue` |
| `event/cdr.go` | `CDRData`: CDR event with topic `cdr.created` |
| `call/v1/inbound.go` | Inbound callback request bodies for callctrl (admit / ring / bridge / hangup) |

Events are published through the unified envelope (`vocara/pkg/eventbus`) with the topic prefix `vocara.event.`.

## Consumers

- Business side: ACD publishes allocation events, and consumers such as push parse them according to the contracts.
- Core side: callctrl subscribes to `queue.dequeue`, publishes `cdr.created`, and receives inbound callbacks as `call/v1` types.
