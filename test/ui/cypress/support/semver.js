Cypress.Commands.add('semver', () => {
  return new Cypress.Promise((resolve, _) => {
    resolve(require('semver'))
  })
})
