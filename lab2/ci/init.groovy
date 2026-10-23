import jenkins.model.Jenkins
import hudson.security.HudsonPrivateSecurityRealm
import hudson.security.FullControlOnceLoggedInAuthorizationStrategy
import org.jenkinsci.plugins.workflow.job.WorkflowJob
import org.jenkinsci.plugins.workflow.cps.CpsFlowDefinition

def j = Jenkins.get()
j.setNumExecutors(2)
def realm = new HudsonPrivateSecurityRealm(false)
realm.createAccount('student', System.getenv('LAB_ADMIN_PASSWORD'))
j.setSecurityRealm(realm)
def auth = new FullControlOnceLoggedInAuthorizationStrategy()
auth.setAllowAnonymousRead(false)
j.setAuthorizationStrategy(auth)
def job = j.getItem('tiopo-lab2') ?: j.createProject(WorkflowJob, 'tiopo-lab2')
def source = new File(System.getenv('LAB_ROOT') + '/lab2/Jenkinsfile').text
job.setDefinition(new CpsFlowDefinition(source, true))
job.save()
j.save()
