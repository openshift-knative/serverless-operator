import OpenshiftConsole from '../../code/openshift/openshiftConsole'

describe('OCP UI for Serverless Eventing', () => {

  const openshiftConsole = new OpenshiftConsole()
  const ns = Cypress.env('TEST_NAMESPACE')

  it('have Eventing bits to add', () => {
    openshiftConsole.login()
    cy.visit(`/add/ns/${ns}`)
    cy.contains('Event Source')
    cy.contains('Event Sink')
  })
})
