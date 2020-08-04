#!/bin/bash

echo "generating sqlite database"
/usr/bin/initializer --manifests=/manifests --output=/bundle/bundles.db --permissive=true