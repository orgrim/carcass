# Carcass

Carcass is a tool to manage sets of libvirt based virtual machines. From
creation to customisation, using libvirt, Terraform, Ansible, CFSSL and a
simple web UI.

This a toy project that will hopefully help me learn more about Go and simplify
the management of the virtual machines I use daily for my job and experiments.

## Installation

Clone the repository and build the tool:

```
$ git clone https://github.com/orgrim/carcass.git
$ cd carcass
$ go install github.com/orgrim/carcass/cmd/carcass
```

CGO must be enabled to build and link against libvirt. This means development
packages for livbirt are required, this is `libvirt-dev` on Debian.

## License

BSD 2-Clause - See [LICENSE][license] file

[license]: https://github.com/orgrim/carcass/blob/master/LICENSE
