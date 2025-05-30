#!/usr/bin/env python3
# Copyright 2025 Ubuntu
# See LICENSE file for licensing details.

"""Go Charm entrypoint."""

import logging
import typing

import ops
import paas_charm.go

logger = logging.getLogger(__name__)

APP_NAME = "ubuntu-insights"
WEB_STATIC_PATH = "/etc/ubuntu-insights-service/web-config.yaml"
WEB_DYNAMIC_PATH = "/etc/ubuntu-insights-service/web-live-config.json"
INGEST_STATIC_PATH = "/etc/ubuntu-insights-service/ingest-config.yaml"
INGEST_DYNAMIC_PATH = "/etc/ubuntu-insights-service/ingest-live-config.json"


class UbuntuInsightsServicesCharm(paas_charm.go.Charm):
    """Go Charm service."""

    def __init__(self, *args: typing.Any) -> None:
        """Initialize the instance.

        Args:
            args: passthrough to CharmBase.
        """
        super().__init__(*args)

        self.pebble_web_service_name = "web-service"
        self.pebble_ingest_service_name = "ingest-service"

        self.framework.observe(self.on.start, self._on_pebble_ready)

    def _on_pebble_ready(self, event: ops.PebbleReadyEvent) -> None:
        container = event.workload
        container.add_layer("ubuntu_insights", self._pebble_layer, combine=True)
        container.replan()
        self.unit.status = ops.ActiveStatus()

    @property
    def _pebble_layer(self) -> ops.pebble.Layer:
        """Pebble layer for the web service."""
        web_command = " ".join(
            [
                "ubuntu-insights-web-service",
                f"--config={WEB_STATIC_PATH}",
                f"--daemon-config={WEB_DYNAMIC_PATH}",
            ]
        )

        ingest_command = " ".join(
            [
                "ubuntu-insights-ingest-service",
                f"--config={INGEST_STATIC_PATH}",
                f"--daemon-config={INGEST_DYNAMIC_PATH}",
            ]
        )

        pebble_layer: ops.pebble.LayerDict = {
            "summary": "{APP_NAME} layer",
            "description": "pebble config layer for Ubuntu Insights server services",
            "services": {
                self.pebble_web_service_name: {
                    "override": "replace",
                    "summary": "web service",
                    "command": web_command,
                    "startup": "enabled",
                },
                self.pebble_ingest_service_name: {
                    "override": "replace",
                    "summary": "ingest service",
                    "command": ingest_command,
                    "startup": "enabled",
                },
            },
        }
        return ops.pebble.Layer(pebble_layer)


if __name__ == "__main__":
    ops.main(UbuntuInsightsServicesCharm)
