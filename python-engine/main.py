import asyncio
import logging
import time
from typing import Any

from nats.aio.client import Client as NATS
from nats.js.client import JetStreamContext
from nats.js.errors import NotFoundError

from pb.trading import events_pb2

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

    logger.info("Requesting exact portfolio state from Go OMS")
    current_exposure: float = 0.0

    try:
        state_msg = await nc.request("oms.state.get", b"", timeout=5.0)
        state = events_pb2.PortfolioState()
        state.ParseFromString(state_msg.data)

        current_exposure = state.balances.get("BTCUSDT", 0.0)
        logger.info(f"State Hydrated. Current BTCUSDT Exposure: {current_exposure}")
    except Exception as e:
        logger.error(
            f"Failed to hydrate state from OMS. Shutting down to prevent desync: {e}"
        )
        await nc.close()
        return

    current_candle_minute: int | None = None
    candle_close_price: float = 0.0
    history_10: list[float] = []
    history_50: list[float] = []

    async def message_handler(msg: Any) -> None:
        nonlocal current_exposure
        nonlocal current_candle_minute, candle_close_price
        nonlocal history_10, history_50

        tick = events_pb2.MarketTick()
        tick.ParseFromString(msg.data)

        tick_minute = tick.event_timestamp_ms // 60000

        if current_candle_minute is None:
            current_candle_minute = tick_minute

        if tick_minute > current_candle_minute:
            history_10.append(candle_close_price)
            history_50.append(candle_close_price)

            if len(history_10) > 10:
                history_10.pop(0)
            if len(history_50) > 50:
                history_50.pop(0)

            if len(history_50) == 50:
                sma10 = sum(history_10) / 10.0
                sma50 = sum(history_50) / 50.0

                is_flat = current_exposure < 0.0001
                is_long = current_exposure >= 0.0001

                target = current_exposure
                reasoning = ""

                if sma10 > sma50 and is_flat:
                    target = 1.0
                    reasoning = f"Golden Cross: SMA10 ({sma10:.2f}) > SMA50 ({sma50:.2f})"
                elif sma10 < sma50 and is_long:
                    target = 0.0
                    reasoning = f"Death Cross: SMA10 ({sma10:.2f}) < SMA50 ({sma50:.2f})"

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

            current_candle_minute = tick_minute

        candle_close_price = tick.price

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
