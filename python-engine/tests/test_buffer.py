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
