import Environment from "./environment";

class Kubernetes {
  ensureNamespaceExists(namespace) {
    const environment = new Environment()
    cy.log(`Ensure namespace ${namespace} exists`)
    this.doesNamespaceExists(namespace).then((exists) => {
      if (!exists) {
        cy.exec(`kubectl create ns ${namespace}`)
        cy.exec(`oc adm policy add-role-to-user edit "${environment.username()}" -n "${namespace}"`)
      }
    })
  }

  doesNamespaceExists(namespace) {
    return new Cypress.Promise((resolve, _) => {
      const cmd = `kubectl get ns ${namespace} -o name`
      cy.exec(cmd, {failOnNonZeroExit: false}).then((result) => {
        let out = result.stdout.trim()
        resolve(out.length > 0)
      })
    })
  }
}

export default Kubernetes
