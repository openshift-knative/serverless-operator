import { describe, it } from 'node:test'
import assert from 'node:assert'
import SemverResolver from './semverResolver.js'

describe('SemverResolver', () => {
  describe('satisfies with <= operator', () => {
    const testCases = [
      // Regular versions
      { version: '4.19.18', range: '<=4.19', expected: true },
      { version: '4.19.0',  range: '<=4.19', expected: true },
      { version: '4.20.0',  range: '<=4.19', expected: false },
      { version: '4.20.4',  range: '<=4.19', expected: false },
      { version: '4.18.0',  range: '<=4.19', expected: true },
      { version: '4.14.0',  range: '<=4.19', expected: true },
      // Pre-release versions
      { version: '4.19.0-rc',     range: '<=4.19', expected: true },
      { version: '4.20.0-rc',     range: '<=4.19', expected: false },
      { version: '4.19.0-alpha2', range: '<=4.19', expected: true },
      { version: '4.20.0-alpha2', range: '<=4.19', expected: false },
      { version: '4.21.0-rc',     range: '<=4.19', expected: false },
    ]

    testCases.forEach(({ version, range, expected }) => {
      const verb = expected ? 'satisfies' : 'does NOT satisfy'
      it(`${version} ${verb} ${range}`, () => {
        assert.strictEqual(
          new SemverResolver(version).satisfies(range),
          expected
        )
      })
    })
  })
})