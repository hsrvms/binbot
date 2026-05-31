import numpy as np
from numpy.typing import NDArray


class RingBuffer:
    def __init__(self, capacity: int) -> None:
        self.capacity: int = capacity
        self.prices: NDArray[np.float64] = np.zeros(capacity, dtype=np.float64)
        self.volumes: NDArray[np.float64] = np.zeros(capacity, dtype=np.float64)

        self.head: int = 0
        self.is_full: bool = False

    def append(self, price: float, volume: float) -> None:
        """O(1) insertion by overwriting data at the current pointer."""
        self.prices[self.head] = price
        self.volumes[self.head] = volume

        self.head += 1
        if self.head >= self.capacity:
            self.head = 0
            self.is_full = True

    def get_latest_price(self) -> float:
        """Retrieves the most recently appended price."""
        idx = self.capacity - 1 if self.head == 0 else self.head - 1
        return float(self.prices[idx])
