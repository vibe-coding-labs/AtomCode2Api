#!/usr/bin/env python3
"""A counter function that returns 1, 2, 3, ... on each call."""


def counter() -> int:
    counter.count += 1
    return counter.count


counter.count = 0


def reset() -> None:
    """Reset the counter so the next call returns 1."""
    counter.count = 0


# Usage demo
if __name__ == "__main__":
    print(counter())  # 1
    print(counter())  # 2
    print(counter())  # 3
    reset()
    print(counter())  # 1 — after reset
    print(counter())  # 2
