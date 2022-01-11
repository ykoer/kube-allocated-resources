# kube-allocated-resources

This is a simple go tool, that calculates the allocated resources from all nodes matching the label selector.

## Build

Build on Linux to run on Linux:

```
go build -o kube-allocated-resources
```

Build on Mac to run on Linux:

```
GOOS=linux GOARCH=386 CGO_ENABLED=0 go build -o kube-allocated-resources
```

## Usage

```
./kube-allocated-resources -h

Usage of ./kube-allocated-resources:
  -d	Return node details
  -g	Return totals grouped by the instance type
  -l string
    	Selector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2) (default "servicecomponent=workernode")
  -o string
    	Output format. One of: json or yaml (default "json")
```


## Examples

### Get the to allocated resources from all nodes matching the label selector

```bash
./kube-allocated-resources -l node-role.kubernetes.io/worker="",node.kubernetes.io/instance-type=m5.2xlarge -o json| jq
```

Output:

```json
{
  "totals": {
    "node_count": 39,
    "cpu_requests": 225652,
    "cpu_requests_percentage": 72,
    "cpu_limits": 666195,
    "cpu_limits_percentage": 213,
    "cpu_total": 312000,
    "memory_requests": 481475656192,
    "memory_requests_percentage": 37,
    "memory_limits": 940615522048,
    "memory_limits_percentage": 72,
    "memory_total": 1291281694720,
    "pods_allocated": 1692,
    "pods_total": 9750,
    "pods_allocated_percentage": 17
  }
}
```

### Get the to allocated resources from nodes matching the label selector detailed per node

```bash
./kube-allocated-resources -l node-role.kubernetes.io/worker="" -o json -d | jq
```

Output:

```json
{
  "totals": {
    "node_count": 45,
    "cpu_requests": 273755,
    "cpu_requests_percentage": 65,
    "cpu_limits": 766395,
    "cpu_limits_percentage": 182,
    "cpu_total": 420000,
    "memory_requests": 543121211136,
    "memory_requests_percentage": 30,
    "memory_limits": 1090504162048,
    "memory_limits_percentage": 60,
    "memory_total": 1791866646528,
    "pods_allocated": 1839,
    "pods_total": 11250,
    "pods_allocated_percentage": 16
  },
  "nodes": [
    {
      "node_name": "ip-10-30-46-119.ec2.internal",
      "instance_type": "m5.2xlarge",
      "cpu_requests": 5959,
      "cpu_requests_percentage": 74.4875,
      "cpu_limits": 10300,
      "cpu_limits_percentage": 128.75,
      "cpu_total": 8000,
      "memory_requests": 9690939392,
      "memory_requests_percentage": 29.16565788753514,
      "memory_limits": 14728298496,
      "memory_limits_percentage": 44.32599336597258,
      "memory_total": 33227227136,
      "pods_allocated": 30,
      "pods_total": 250,
      "pods_allocated_percentage": 12
    },
    ...
  ]
}
```

### Get the to allocated resources from nodes matching the label selector grouped by the instance type

```bash
./kube-allocated-resources -l node-role.kubernetes.io/worker="" -o json -g | jq
```

Output:

```
{
  "totals": {
    "node_count": 45,
    "cpu_requests": 273830,
    "cpu_requests_percentage": 65,
    "cpu_limits": 767895,
    "cpu_limits_percentage": 182,
    "cpu_total": 420000,
    "memory_requests": 544731823872,
    "memory_requests_percentage": 30,
    "memory_limits": 1092114774784,
    "memory_limits_percentage": 60,
    "memory_total": 1791866646528,
    "pods_allocated": 1842,
    "pods_total": 11250,
    "pods_allocated_percentage": 16
  },
  "groupedby_instance_type": [
    {
      "node_count": 3,
      "instance_type": "m5.8xlarge",
      "cpu_requests": 44992,
      "cpu_requests_percentage": 46,
      "cpu_limits": 94200,
      "cpu_limits_percentage": 98,
      "cpu_total": 96000,
      "memory_requests": 53357510656,
      "memory_requests_percentage": 13,
      "memory_limits": 146569700352,
      "memory_limits_percentage": 36,
      "memory_total": 400900472832,
      "pods_allocated": 60,
      "pods_total": 750,
      "pods_allocated_percentage": 8
    },
    {
      "node_count": 39,
      "instance_type": "m5.2xlarge",
      "cpu_requests": 224717,
      "cpu_requests_percentage": 72,
      "cpu_limits": 667695,
      "cpu_limits_percentage": 214,
      "cpu_total": 312000,
      "memory_requests": 479615581952,
      "memory_requests_percentage": 37,
      "memory_limits": 939253618432,
      "memory_limits_percentage": 72,
      "memory_total": 1291281694720,
      "pods_allocated": 1696,
      "pods_total": 9750,
      "pods_allocated_percentage": 17
    },
    {
      "node_count": 3,
      "instance_type": "r5.xlarge",
      "cpu_requests": 4121,
      "cpu_requests_percentage": 34,
      "cpu_limits": 6000,
      "cpu_limits_percentage": 50,
      "cpu_total": 12000,
      "memory_requests": 11758731264,
      "memory_requests_percentage": 11,
      "memory_limits": 6291456000,
      "memory_limits_percentage": 6,
      "memory_total": 99684478976,
      "pods_allocated": 86,
      "pods_total": 750,
      "pods_allocated_percentage": 11
    }
  ]
}
```


### JQ query examples

#### Get tab delimited values

```bash
./kube-allocated-resources -l node-role.kubernetes.io/worker="",node.kubernetes.io/instance-type=m5.8xlarge -o json | jq -r '[
	.totals.cpu_total,
	.totals.cpu_requests
] | @tsv'
```

Output:

```
96000	44992
```

#### Do some math

```bash
./kube-allocated-resources -l node-role.kubernetes.io/worker="",node.kubernetes.io/instance-type=m5.8xlarge -o json -d | jq -r '. | [
        reduce .nodes[].cpu_total as $num (0; .+$num),
        reduce .nodes[].cpu_requests as $num (0; .+$num)
        ] | @tsv'
```
Output:

```
96000	44992
```
