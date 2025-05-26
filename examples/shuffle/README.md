# Shuffle Operator Examples

The `shuffle` operator randomly reorders elements in arrays. It's useful for load balancing, randomizing test data, and distributing resources across availability zones.

## Files in this directory:

1. **basic.yml** - Basic array shuffling
2. **availability-zones.yml** - AZ distribution for load balancing
3. **test-data.yml** - Randomizing test data sets
4. **load-balancing.yml** - Server and task distribution

## Key Features:

- Randomly reorders array elements
- Works with any array type (strings, numbers, objects)
- Different output each time (non-deterministic)
- Can shuffle multiple arrays into one
- Useful for pseudo-random distribution

## Common Use Cases:

```yaml
# Shuffle a list
shuffled: (( shuffle original_list ))

# Combine and shuffle multiple lists
combined: (( shuffle list1 list2 list3 ))

# Distribute across AZs
instance:
  az: (( grab (shuffle availability_zones).0 ))
```

## Running Examples:

```bash
# Basic shuffling (run multiple times to see different outputs)
spruce merge basic.yml
spruce merge basic.yml  # Different order

# AZ distribution
spruce merge availability-zones.yml

# Test data generation
spruce merge test-data.yml
```

## Notes:

- Output is non-deterministic (changes each run)
- Useful for approximating load distribution
- Not cryptographically secure randomness
- Good for development and testing scenarios