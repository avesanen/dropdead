Dropdead
========

Dropdead is a small drag&drop image sharing service.


Requirements
============

- go v.1.8+ (uses http shutdown)


Config Example
==============

```yaml
addr: 127.0.0.1:8000
db_path: /var/dropdead
uploads_path: /var/dropdead
```

Dropdead will create an `bolt.db` file under db_path.

Uploads will be saved `uploads` directory under the uploads_path.


Usage
=====

Running dropdead without parameters will create a data directory to the current location:

`./dropdead`


To change this behavior, run dropdead with `-c <config file>` option:

`./dropdead -c /etc/config.yaml`
