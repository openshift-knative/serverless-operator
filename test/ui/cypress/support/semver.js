const semver = require('semver')

class SemverResolver {
  constructor(version) {
    this.version = semver.coerce(version)
  }

  satisfies(range) {
    const r = new semver.Range(range)
    const result = r.test(this.version)
    console.log(`Version '${this.version}' matches range '${range}': ${result}`)
    return result
  }
}

Cypress.Commands.add('semver', (version) => {
  return new Cypress.Promise((resolve, _) => {
    resolve(new SemverResolver(version))
  })
})
