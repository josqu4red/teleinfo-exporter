# teleinfo-exporter

Prometheus exporter for french electricity meter telemetry.

Exposes power/intensity metrics from a computer connected to an electricity meter through serial link.

# Hardware requirements

Data can be gathered from the meter with a small GPIO module mounted on a Raspberry Pi or similar small device.
[More detail (in french)](https://hallard.me/pitinfo/).

# Serial data format

Protocol and data are described exhaustively in [this document](https://www.enedis.fr/media/2027/download).
