# provisionize
Zero touch provisioning for oVirt VMs with Google Cloud DNS integration

## Remarks
Since this is an early develpoment version, there can be breaking changes before reaching version 1.0.

## Client

### Installation
```bash
go get github.com/mauvesoftware/provisionize/cmd/provisionizer
```

### Usage
```bash
./provisionizer --id=foo --cluster=cluster1 --cores=2 --memory=2048 --template=ubuntu-18-04 --ipv4=10.2.3.4 --ipv6=2001:678:1e0:f00::1 test-vm
```

## Server

### Installation
To build and install provisionize on your local system (without usind docker)

```bash
go get github.com/mauvesoftware/provisionize/cmd/provisionize
```

### Configuration

this is an example config file for the ovirt server
```yaml
listen_address: "[::]:1337"
ovirt:
  url: https://my-ovirt.instance
  username: provisionize
  password: allTheThings
  template_path: /etc/provisionize/template
gcloud:
  project_id: "123456"
```

An example how /etc/provisionize/template can look like can be found in `examples/template.xml`

### Running in Docker
Assuming that your config file is located under /etc/provisionize/config.yml and we want to expose the gRPC port 1337:

```bash
docker run -d --restart=always -v /etc/provisionize/config.yml:/config/config.yml -p 1337:1337 mauvesoftware/provisionize
```

### Running the binary
```bash
provisionize -c /etc/provisionize/config.yml
```

## Authors
[Daniel Czerwonk (Mauve Mailorder Software)]( https://github.com/czerwonk )

## License
(c) Mauve Mailorder Software, 2019. Licensed under MIT license.
