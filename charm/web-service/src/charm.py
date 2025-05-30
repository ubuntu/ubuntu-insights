#!/usr/bin/env python3
# Copyright 2025 Ubuntu
# See LICENSE file for licensing details.

"""Go Charm entrypoint."""

import logging
import typing

import ops
import paas_charm.go
from ops.model import ActiveStatus, SecretNotFoundError
from ops.pebble import LayerDict

logger = logging.getLogger(__name__)

APP_NAME = "ubuntu-insights-web-service"
STATIC_PATH = "/etc/ubuntu-insights-web-service/config.yaml"
DYNAMIC_PATH = "/data/live-config.json"


class UbuntuInsightsWebServicesCharm(paas_charm.go.Charm):
    """Go Charm service."""

    def __init__(self, *args: typing.Any) -> None:
        """Initialize the instance.

        Args:
            args: passthrough to CharmBase.
        """
        super().__init__(*args)

        self.framework.observe(self.on.install, self._on_event)
        self.framework.observe(self.on.start, self._on_event)

    def _on_static_secret(self, event):
        self._write_secret_file(event.relation, "static-config", STATIC_PATH)
        self._maybe_start_service()

    def _on_dynamic_secret(self, event):
        self._write_secret_file(event.relation, "dynamic-config", DYNAMIC_PATH)
        # No need to restart the service

    def _on_event(self, event):
        self._maybe_start_service()

    def _write_secret_file(self, relation, key, target_path):
        try:
            secret = relation.get_secret(scope="app")
            content = secret.get_content().get(key)
            container = self.unit.get_container("my-container")
            if container.can_connect():
                container.push(target_path, content, make_dirs=True)
        except SecretNotFoundError:
            return

    def _maybe_start_service(self):
        container = self.unit.get_container("my-container")
        if not container.can_connect():
            return

        # Check if both config files exist
        try:
            container.pull(STATIC_PATH)
            container.pull(DYNAMIC_PATH)
        except FileNotFoundError:
            return  # Wait until files are present
        # Add Pebble layer
        layer: LayerDict = {
            "summary": f"{APP_NAME} layer",
            "services": {
                "myapp": {
                    "override": "replace",
                    "command": (
                        f"/usr/bin/{APP_NAME} "
                        f"--config={STATIC_PATH} "
                        f"--daemon-config={DYNAMIC_PATH}"
                    ),
                    "startup": "enabled",
                }
            },
        }
        container.add_layer(APP_NAME, layer, combine=True)
        container.autostart()
        self.unit.status = ActiveStatus("Running")


if __name__ == "__main__":
    ops.main(UbuntuInsightsWebServicesCharm)
