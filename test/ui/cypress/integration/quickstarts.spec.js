describe('OCP UI for Serverless', () => {
  it('has Serverless quickstarts', () => {
    cy.login()
    cy.visit('/quickstart')
    cy.contains('Exploring Serverless applications')
  })
})
