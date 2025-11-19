import chalk from 'chalk'
import {applyPatches as diffApplyPatches} from 'diff'
import fs from 'fs/promises'
import path from 'path'
import cachedir from 'cachedir'
import {createRequire} from "module"

function getVersionDir() {
  const require = createRequire(import.meta.url)
  const cypressPackageJson = require('cypress/package.json')
  const cacheDir = cachedir('Cypress')
  const version = cypressPackageJson.version
  return path.join(cacheDir, version)
}

async function applyPatches() {
  const patchesDir = path.join('cypress', 'patches')
  const patchesAbsDir = path.join(process.cwd(), patchesDir)
  const patches = await fs.readdir(patchesAbsDir)
  const installDir = getVersionDir()

  console.log(`\n> Applying patches on to ${chalk.cyan(installDir)}\n`)

  for (const filename of patches) {
    if (!filename.endsWith('.patch')) {
      continue;
    }
    const fullpath = path.join(patchesAbsDir, filename)
    const enc = 'utf8'
    const patch = await fs.readFile(fullpath, enc)
    const relativeFilename = path.join(patchesDir, filename)
    console.log(`>> Applying patch ${chalk.cyan(relativeFilename)}`)
    await diffApplyPatches(patch, {
      loadFile: (index, callback) => {
        console.debug(`>>> Loading old file: ${chalk.red(index.oldFileName)}`)
        fs.readFile(path.join(installDir, index.oldFileName), enc)
          .then((contents) => {
            callback(null, contents)
          }).catch(callback)
      },
      patched: (index, content, callback) => {
        if (content === false) {
          console.debug(`>>> Already patched: ${chalk.yellow(index.newFileName)}`)
          return callback()
        }
        console.debug(`>>> Patched new file: ${chalk.green(index.newFileName)}`)
        fs.writeFile(path.join(installDir, index.newFileName), content, enc)
          .then(callback)
          .catch(callback)
      },
      complete: (err) => {
        if (err !== undefined) {
          throw err
        }
        console.log(`>> Successfully applied patch ${chalk.cyan(relativeFilename)}`)
      }
    })
  }
}

applyPatches().catch(e => {
  console.error(e)
  process.exit(42)
})
