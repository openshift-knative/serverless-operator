import OpenshiftConsole from "../code/openshift/openshiftConsole";

describe('OCP UI for Serverless', () => {

  beforeEach(() => {
    const openShiftConsole = new OpenshiftConsole()
    openShiftConsole.login()
  })

  it('has Serverless quickstarts', () => {
    cy.visit('/quickstart')
    cy.contains('Exploring Serverless applications')
  })
})
