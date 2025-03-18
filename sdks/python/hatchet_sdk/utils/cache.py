from collections import OrderedDict
from typing import Generic, TypeVar

K = TypeVar("K")
V = TypeVar("V")


class LRUCache(Generic[K, V]):
    def __init__(self, capacity: int) -> None:
        self.cache = OrderedDict[K, V]()
        self.capacity = capacity

    def get(self, key: K) -> V | None:
        if key not in self.cache.keys():
            return None
        else:
            self.cache.move_to_end(key)
            return self.cache[key]

    def put(self, key: K, value: V) -> None:
        self.cache[key] = value
        self.cache.move_to_end(key)
        if len(self.cache) > self.capacity:
            self.cache.popitem(last=False)

    def evict(self, key: K) -> None:
        if key in self.cache.keys():
            self.cache.pop(key)
