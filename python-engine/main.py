import asyncio
import logging

from nats.aio.client import Client as NATS
from pb.trading import events_pb2
from strategy.buffer import RingBuffer

logging.basicConfi(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(message)s")
logger = logging.getLogger("PythonEngine")


async def main() -> None:
    nc = NATS()
    await nc.connect("nats://nats:4222")
    js = nc.jetstream()

    buffer = RingBuffer(capacity=1000)

    async def message_handler(msg) -> None:
        tick = events_pb2.MarketTick()
        tick.ParseFromString(msg.data)

        buffer.append(price=tick.price, volume=tick.volume)

        if buffer.head % 100 == 0:
            logger.info(
                f"Decoded -> Symbol: {tick.symbol}, Price: {tick.price}, Vol: {tick.volume}"
            )

        await msg.ack

    sub = await js.subscribe(
        subject="market.data.BTCUSDT",
        durable="PYTHON_ENGINE_CONSUMER",
        cb=message_handler,
    )

    logger.info("Python Engine initialized. Listening for binary market data...")

    try:
        while True:
            await asyncio.sleep(1)
    except asyncio.asyncio.CancelledError:
        pass
    finally:
        await sub.unsubscribe()
        await nc.close()

if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        logger.info("Gracefully shutting down Python Engine.")
