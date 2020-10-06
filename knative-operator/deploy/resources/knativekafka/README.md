- Download the files from Github releases (no auto download at the moment)
- Remove comments and blank lines with
```
sed -i \
        -e '/^[ \t]*#/d' \
        -e '/^[ \t]*$/d' \
        kafkachannel-vX.XX.X.yaml

sed -i \
        -e '/^[ \t]*#/d' \
        -e '/^[ \t]*$/d' \
        kafkasource-vX.XX.X.yaml
```
- Create `kafkachannel-latest.yaml` and `kafkasource-latest.yaml` symlinks for the new files
- Dump the old files
