import OpenshiftConsole from '../code/openshift/openshiftConsole'

describe('OCP UI for Serverless', () => {

  const openShiftConsole = new OpenshiftConsole()

  it('has Serverless quickstarts', () => {
    openShiftConsole.login()
    cy.visit('/quickstart')
    cy.contains('Exploring Serverless applications')
  })
})
