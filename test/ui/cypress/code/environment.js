import SemverResolver from "./semverResolver";

class Environment {
  static #rand = undefined

  ocpVersion() {
    return new SemverResolver(Cypress.env('OCP_VERSION'))
  }

  loginProvider() {
    return Cypress.env('OCP_LOGIN_PROVIDER')
  }

  username() {
    return Cypress.env('OCP_USERNAME')
  }

  password() {
    return Cypress.env('OCP_PASSWORD')
  }

  namespace() {
    return Cypress.env('TEST_NAMESPACE')
  }

  random() {
    if (Environment.#rand === undefined) {
      let seed = Cypress.env('TEST_RANDOM')
      if (seed === undefined) {
        seed = (Math.random() + 1).toString(36).substring(7)
      }
      cy.log(`Using seed: ${seed} in testing.`)
      cy.log(`Set TEST_RANDOM=${seed} to recreate this test.`)
      Environment.#rand = new PRNG(seed)
    }
    return Environment.#rand
  }
}

class PRNG {
  #rand

  constructor(seedString) {
    // Create xmur3 state:
    const seed = xmur3(seedString);
    // Output four 32-bit hashes to provide the seed for sfc32.
    this.#rand = sfc32(seed(), seed(), seed(), seed())
  }

  next() {
    return this.#rand()
  }
}

// See: https://stackoverflow.com/a/47593316/844449
function xmur3(str) {
  for (var i = 0, h = 1779033703 ^ str.length; i < str.length; i++) {
    h = Math.imul(h ^ str.charCodeAt(i), 3432918353);
    h = h << 13 | h >>> 19;
  }
  return function () {
    h = Math.imul(h ^ (h >>> 16), 2246822507);
    h = Math.imul(h ^ (h >>> 13), 3266489909);
    return (h ^= h >>> 16) >>> 0;
  }
}

function sfc32(a, b, c, d) {
  return function () {
    a >>>= 0;
    b >>>= 0;
    c >>>= 0;
    d >>>= 0;
    var t = (a + b) | 0;
    a = b ^ b >>> 9;
    b = c + (c << 3) | 0;
    c = (c << 21 | c >>> 11);
    d = d + 1 | 0;
    t = t + d | 0;
    c = c + t | 0;
    return (t >>> 0) / 4294967296;
  }
}

export default Environment
