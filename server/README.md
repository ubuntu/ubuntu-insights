# Ubuntu Insights Server Services

The Ubuntu Insights Services are the components for aggregating and processing reports. They do not record any personally identifying information such as IPs.

There are two server services, a web exposed web service which handles incoming HTTP requests as well as an ingest service which does simple validations before inserting reports into a database.

Neither of these services are meant for local use.