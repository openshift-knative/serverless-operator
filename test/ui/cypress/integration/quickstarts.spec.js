describe('OCP UI for Serverless', () => {
  it('have Serverless quickstarts', () => {
    cy.login()
    cy.visit('/quickstart')
    cy.contains('Exploring Serverless applications')
  })
})
