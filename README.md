# load-balancer-demo

Simple implementation of a Load Balancer in Go.

## Backend Server `be`

### How to run

```sh
$ make build
$ ./bin/be -port=8081
```

This will start the backend server on port `8081`.

## Load Balancer `lb`

### How to run

```sh
$ make build
$ ./bin/lb -port=80 -backends=localhost:8081,localhost:8082
```

This will start the Load Balancer on port `80`, distributing traffic between the backends on ports `8081` and `8082`.

## Testing

Invoke `curl` to make concurrent requests:

```sh
$ curl --parallel --parallel-immediate --parallel-max 3 --config urls.txt
```

## References

- [Coding Challenges: Build You Own Load Balancer](https://codingchallenges.fyi/challenges/challenge-load-balancer)
