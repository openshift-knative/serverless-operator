import semver from 'semver'

class SemverResolver {
  constructor(version) {
    this.version = semver.coerce(version)
  }

  satisfies(range) {
    const r = new semver.Range(range)
    return r.test(this.version)
  }
}

export default SemverResolver
