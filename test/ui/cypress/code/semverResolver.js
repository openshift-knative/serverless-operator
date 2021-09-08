import semver from 'semver'

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

export default SemverResolver
