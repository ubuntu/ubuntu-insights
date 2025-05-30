#!/usr/bin/env python3
# Copyright 2025 Ubuntu
# See LICENSE file for licensing details.

"""Go Charm entrypoint."""

import logging
import typing

import ops
from charms.data_platform_libs.v0.data_interfaces import DatabaseCreatedEvent, DatabaseRequires

logger = logging.getLogger(__name__)

APP_NAME = "ubuntu-insights"
WEB_STATIC_PATH = "/etc/ubuntu-insights-service/web-config.yaml"
WEB_DYNAMIC_PATH = "/etc/ubuntu-insights-service/web-live-config.json"
INGEST_STATIC_PATH = "/etc/ubuntu-insights-service/ingest-config.yaml"
INGEST_DYNAMIC_PATH = "/etc/ubuntu-insights-service/ingest-live-config.json"


class UbuntuInsightsServicesCharm(ops.CharmBase):
    """Go Charm service."""

    def __init__(self, *args: typing.Any) -> None:
        """Initialize the instance.

        Args:
            args: passthrough to CharmBase.
        """
        super().__init__(*args)

        self.pebble_web_service_name = "web-service"
        self.pebble_ingest_service_name = "ingest-service"

        # The 'relation_name' comes from the 'charmcraft.yaml file'.
        # The 'database_name' is the name of the database that our application requires.
        self.database = DatabaseRequires(self, relation_name="database", database_name="insights")

        # See https://charmhub.io/data-platform-libs/libraries/data_interfaces
        self.framework.observe(self.database.on.database_created, self._on_database_created)
        self.framework.observe(self.database.on.endpoints_changed, self._on_database_created)

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
                    "environment": self.ingest_environment,
                },
            },
        }
        return ops.pebble.Layer(pebble_layer)

    @property
    def ingest_environment(self) -> dict[str, str]:
        """Environment variables for the ingest service."""
        db_data = self.fetch_postgres_relation_data()
        if not db_data:
            return {}
        env = {
            key: value
            for key, value in {
                "DEMO_SERVER_DB_HOST": db_data.get("db_host", None),
                "DEMO_SERVER_DB_PORT": db_data.get("db_port", None),
                "DEMO_SERVER_DB_USER": db_data.get("db_username", None),
                "DEMO_SERVER_DB_PASSWORD": db_data.get("db_password", None),
            }.items()
            if value is not None
        }
        return env

    def _on_database_created(self, event: DatabaseCreatedEvent) -> None:
        # Handle the created database.
        # Need to restart the ingest service to pick up the new database connection,
        # using the new database credentials.

        self._update_layer_and_restart()

    def _update_layer_and_restart(self) -> None:
        ops.MaintenanceStatus("Assembling Pebble layers")

    def fetch_postgres_relation_data(self) -> dict[str, str]:
        """Fetch postgres relation data.

        This function retrieves relation data from a postgres database using
        the `fetch_relation_data` method of the `database` object. The retrieved data is
        then logged for debugging purposes, and any non-empty data is processed to extract
        endpoint information, username, and password. This processed data is then returned as
        a dictionary. If no data is retrieved, the unit is set to waiting status and
        the program exits with a zero status code.
        """
        relations = self.database.fetch_relation_data()
        logger.debug("Got following database data: %s", relations)
        for data in relations.values():
            if not data:
                continue
            logger.info("New PSQL database endpoint is %s", data["endpoints"])
            host, port = data["endpoints"].split(":")
            db_data = {
                "db_host": host,
                "db_port": port,
                "db_username": data["username"],
                "db_password": data["password"],
            }
            return db_data
        return {}


if __name__ == "__main__":
    ops.main(UbuntuInsightsServicesCharm)
