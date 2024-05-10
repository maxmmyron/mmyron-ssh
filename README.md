# ssh.mmyron.com

a fork of mmyron.com accessible via `ssh`

## running locally

clone the repo:

```
git clone https://github.com/maxmmyron/mmyron-ssh
cd mmyron-ssh
```

and open a new server with

```
go run .
```

If running for the first time, a set of keys will be generated in the `./.ssh/` folder.
The program will create a new SSH server accessible from `localhost:22`.

## known issues

- client light/dark mode cannot be detected (AFAIK)
- there may be occasional rendering issues when resizing (i think some of these are fixed for the most part but i'm not 100% on this.)

## in the future

- [ ] hosting posts (these prob. need to be pretty heavily reformatted)
