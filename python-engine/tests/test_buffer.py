from strategy.buffer import RingBuffer


def test_ring_buffer_insertion() -> None:
    buffer = RingBuffer(capacity=3)
    assert buffer.is_full is False

    buffer.append(price=100.0, volume=1.0)
    buffer.append(price=101.0, volume=2.0)
    buffer.append(price=102.0, volume=3.0)

    assert buffer.is_full is True
    assert buffer.get_latest_price() == 102.0

    buffer.append(price=103.0, volume=4.0)

    assert buffer.prices[0] == 103.0
    assert buffer.get_latest_price() == 103.0

def test_get_sma() -> None:
    buffer = RingBuffer(capacity=5)

    buffer.append(10.0, 1.0)
    buffer.append(20.0, 1.0)
    assert buffer.get_sma(3) == 0.0

    buffer.append(30.0, 1.0)
    assert buffer.get_sma(3) == 20.0  # (10 + 20 + 30) / 3

    buffer.append(40.0, 1.0)
    buffer.append(50.0, 1.0)

    buffer.append(60.0, 1.0)
    assert buffer.get_sma(3) == 50.0
