package appdefinition

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/ibuildthecloud/baaah/pkg/meta"
	"github.com/ibuildthecloud/baaah/pkg/router"
	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
	"github.com/ibuildthecloud/herd/pkg/condition"
	"github.com/ibuildthecloud/herd/pkg/labels"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

func JobStatus(req router.Request, resp router.Response) error {
	app := req.Object.(*v1.AppInstance)
	cond := condition.Setter(app, resp, v1.AppInstanceConditionJobs)
	jobs := &batchv1.JobList{}

	err := req.Client.List(jobs, &meta.ListOptions{
		Namespace: app.Status.Namespace,
		Selector: klabels.SelectorFromSet(map[string]string{
			labels.HerdManaged: "true",
			labels.HerdAppName: app.Name,
		}),
	})
	if err != nil {
		return err
	}

	var (
		running     bool
		runningName string
		failed      bool
		failedName  string
	)

	sort.Slice(jobs.Items, func(i, j int) bool {
		return jobs.Items[i].Name < jobs.Items[j].Name
	})
	for _, job := range jobs.Items {
		if app.Status.JobsStatus == nil {
			app.Status.JobsStatus = map[string]v1.JobStatus{}
		}

		_, messages, err := podsStatus(req, app.Status.Namespace, klabels.SelectorFromSet(map[string]string{
			labels.HerdManaged: "true",
			labels.HerdJobName: job.Name,
		}))
		if err != nil {
			return err
		}

		jobStatus := v1.JobStatus{
			Message: strings.Join(messages, "; "),
		}
		if job.Status.Active > 0 {
			jobStatus.Running = true
			running = true
			runningName = job.Name
		}
		if job.Status.Failed > 0 {
			jobStatus.Failed = true
			failed = true
			failedName = job.Name
		}
		if job.Status.Succeeded > 0 {
			jobStatus.Succeed = true
		}
		app.Status.JobsStatus[job.Name] = jobStatus
	}

	switch {
	case failed:
		cond.Error(fmt.Errorf("%s: failed [%s]", failedName, app.Status.JobsStatus[failedName].Message))
	case running:
		cond.Unknown(fmt.Sprintf("%s: running [%s]", runningName, app.Status.JobsStatus[runningName].Message))
	default:
		cond.Success()
	}

	resp.Objects(app)
	return nil
}

func podsStatus(req router.Request, namespace string, sel klabels.Selector) (bool, []string, error) {
	var (
		isTransition bool
		message      []string
		pods         = &corev1.PodList{}
	)
	err := req.Client.List(pods, &meta.ListOptions{
		Namespace: namespace,
		Selector:  sel,
	})
	if err != nil {
		return false, nil, err
	}

	for _, pod := range pods.Items {
		for _, cond := range pod.Status.Conditions {
			if cond.Type == corev1.PodScheduled {
				if cond.Status != corev1.ConditionTrue {
					isTransition = true
					message = append(message, podName(&pod)+" is not scheduled to a node")
				}
			}
		}

		msg, transition := containerMessages(&pod, pod.Status.InitContainerStatuses)
		message = append(message, msg...)
		if transition {
			isTransition = true
		}

		msg, transition = containerMessages(&pod, pod.Status.ContainerStatuses)
		message = append(message, msg...)
		if transition {
			isTransition = true
		}
	}

	return isTransition, message, nil
}

func AppStatus(req router.Request, resp router.Response) error {
	app := req.Object.(*v1.AppInstance)
	cond := condition.Setter(app, resp, v1.AppInstanceConditionContainers)
	deps := &appsv1.DeploymentList{}

	err := req.Client.List(deps, &meta.ListOptions{
		Namespace: app.Status.Namespace,
		Selector: klabels.SelectorFromSet(map[string]string{
			labels.HerdManaged: "true",
			labels.HerdAppName: app.Name,
		}),
	})
	if err != nil {
		return err
	}

	notJob, err := klabels.NewRequirement(labels.HerdContainerName, selection.Exists, nil)
	if err != nil {
		return err
	}

	isTransition, message, err := podsStatus(req, app.Status.Namespace, klabels.SelectorFromSet(map[string]string{
		labels.HerdManaged: "true",
		labels.HerdAppName: app.Name,
	}).Add(*notJob))
	if err != nil {
		return err
	}

	container := map[string]v1.ContainerStatus{}
	for _, dep := range deps.Items {
		status := container[dep.Labels[labels.HerdContainerName]]
		status.Ready = dep.Status.ReadyReplicas
		status.ReadyDesired = dep.Status.Replicas
		status.UpToDate = dep.Status.UpdatedReplicas
		container[dep.Labels[labels.HerdContainerName]] = status

		if status.Ready != status.ReadyDesired {
			isTransition = true
			message = append(message, dep.Labels[labels.HerdAppName]+" is not ready")
		}
	}
	app.Status.ContainerStatus = container

	if isTransition {
		sort.Strings(message)
		cond.Unknown(strings.TrimSpace(strings.Join(message, "; ")))
	} else {
		cond.Success()
	}

	if !isTransition && app.Spec.Stop != nil && *app.Spec.Stop {
		allZero := true
		for _, v := range app.Status.ContainerStatus {
			if v.ReadyDesired != 0 {
				allZero = false
				break
			}
		}
		if allZero {
			app.Status.Stopped = true
		}
	} else {
		app.Status.Stopped = false
	}

	resp.Objects(app)
	return nil
}

func containerMessages(pod *corev1.Pod, status []corev1.ContainerStatus) (message []string, isTransition bool) {
	for _, container := range status {
		if container.State.Waiting != nil && container.State.Waiting.Reason != "" {
			isTransition = true
			if container.State.Waiting.Message == "" {
				message = append(message, podName(pod)+" "+
					container.State.Waiting.Reason)
			} else {
				message = append(message, podName(pod)+" "+
					container.State.Waiting.Reason+": "+container.State.Waiting.Message)
			}
		}
		if container.State.Terminated != nil && container.State.Terminated.ExitCode > 0 {
			isTransition = true
			message = append(message, podName(pod)+" "+container.State.Terminated.Reason+": Exit Code "+
				strconv.Itoa(int(container.State.Terminated.ExitCode)))
		}
	}
	return
}

func podName(pod *corev1.Pod) string {
	jobName := pod.Labels[labels.HerdJobName]
	if jobName != "" {
		return jobName
	}
	return pod.Labels[labels.HerdContainerName]
}