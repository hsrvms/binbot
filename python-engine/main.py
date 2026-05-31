import asyncio
import logging
import time
from typing import Any

from nats.aio.client import Client as NATS
from nats.js.client import JetStreamContext
from nats.js.errors import NotFoundError

from pb.trading import events_pb2
from strategy.buffer import RingBuffer

logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(message)s")
logger = logging.getLogger("PythonEngine")


async def main() -> None:
    nc = NATS()
    await nc.connect("nats://nats:4222")
    js: JetStreamContext = nc.jetstream()  # pyright: ignore[reportUnknownMemberType]

    try:
        await js.stream_info("STRATEGY")
    except NotFoundError:
        logger.info("Stream STRATEGY not found, provisioning...")
        await js.add_stream(name="STRATEGY", subjects=["strategy.intent"])  # pyright: ignore[reportUnknownMemberType]

    buffer = RingBuffer(capacity=1000)

    current_exposure: float = 0.0

    async def message_handler(msg: Any) -> None:
        nonlocal current_exposure

        tick = events_pb2.MarketTick()
        tick.ParseFromString(msg.data)

        buffer.append(price=tick.price, volume=tick.volume)

        short_sma = buffer.get_sma(10)
        long_sma = buffer.get_sma(50)

        if short_sma > 0.0 and long_sma > 0.0:
            target = current_exposure
            reasoning = ""

            if short_sma > long_sma and current_exposure == 0.0:
                target = 1.0
                reasoning = (
                    f"Golden Cross: SMA10 ({short_sma:.2f}) > SMA50 ({long_sma:.2f})"
                )

            elif short_sma < long_sma and current_exposure == 1.0:
                target = 0.0
                reasoning = (
                    f"Death Cross: SMA10 ({short_sma:.2f}) < SMA50 ({long_sma:.2f})"
                )

            if target != current_exposure:
                current_exposure = target

                intent = events_pb2.IntentSignal(
                    symbol=tick.symbol,
                    target_exposure=target,
                    strategy_reasoning=reasoning,
                    signal_timestamp_ms=int(time.time() * 1000),
                )

                await js.publish("strategy.intent", intent.SerializeToString())
                logger.info(
                    f"Published IntentSignal -> Target: {target} | Reason: {reasoning}"
                )

        await msg.ack()

    sub = await js.subscribe(
        subject="market.data.BTCUSDT",
        durable="PYTHON_ENGINE_CONSUMER",
        cb=message_handler,
    )

    logger.info("Python Engine initialized. Listening for binary market data...")

    try:
        while True:
            await asyncio.sleep(1)
    except asyncio.CancelledError:
        pass
    finally:
        await sub.unsubscribe()
        await nc.close()


if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        logger.info("Gracefully shutting down Python Engine.")
