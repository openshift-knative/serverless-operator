describe('OCP UI for Serverless', () => {

  beforeEach(() => {
    cy.login()
  })

  it('has Serverless quickstarts', () => {
    cy.visit('/quickstart')
    cy.contains('Exploring Serverless applications')
  })
})
