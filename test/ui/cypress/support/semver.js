const semver = require('semver')

class SemverResolver {
  constructor(version) {
    this.version = version
  }

  satisfies(range) {
    const r = new semver.Range(range)
    return r.test(this.version)
  }
}

Cypress.Commands.add('semver', (version) => {
  return new Cypress.Promise((resolve, _) => {
    resolve(new SemverResolver(version))
  })
})
