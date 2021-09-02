Cypress.Commands.add('ocpVersion', () => {
  return cy.semver(Cypress.env('OCP_VERSION'))
})
