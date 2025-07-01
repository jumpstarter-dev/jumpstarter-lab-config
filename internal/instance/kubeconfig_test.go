package instance

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
)

const (
	validKubeconfig = `
apiVersion: v1
kind: Config
clusters:
- name: test-cluster
  cluster:
    server: https://test-cluster.example.com
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURQRENDQWlTZ0F3SUJBZ0lJVFJIb25HQzdxRGd3RFFZSktvWklodmNOQVFFTEJRQXdKakVTTUJBR0ExVUUKQ3hNSmIzQmxibk5vYVdaME1SQXdEZ1lEVlFRREV3ZHliMjkwTFdOaE1CNFhEVEkxTURZeU16RTJNREkxTWxvWApEVE0xTURZeU1URTJNREkxTWxvd0pqRVNNQkFHQTFVRUN4TUpiM0JsYm5Ob2FXWjBNUkF3RGdZRFZRUURFd2R5CmIyOTBMV05oTUlJQklqQU5CZ2txaGtpRzl3MEJBUUVGQUFPQ0FROEFNSUlCQ2dLQ0FRRUF2WkxDK0FCbFpzWVcKcEpHNUFRYW9PbXlKSDRuSUl3dENiU0dyN1hkQjN6bFZ4aWJITURRWWQ3ZEFiTXc1d1EvalhETlV6OER4RUZKNQpxUjR0SVZRelVSb05tOU9NSU10Rkg2SGhKeFpCVGhSZitmYXY3amh4QVZ3elBlbTV0L09ad2VURHdtdkdoOVNYCkkxUTVUUFVlOHZIaDNPU095TTJTV3Qvckc4UmZhazlucUwydEc5cWhkSkN6Q0xMRGpxdDlZeXJmbUIzbHo0ZUUKcTFJMXViZGVWVlpadjJHWXN6Wll6NTVBOGYxell6aE4rZHVEQ0s2WlhrZVozOFNuY2hHRXFqd21MN3VBK1RkaQpScW9ESlZkRTRmV0lvUTEzUXNBcytWMXF2Q21rWHpOWWxKWFlqMHBXU1dyU3hpaFdTd0lWaG0zRXhuRnhYYnFqCkNFdm1iT3dIeFFJREFRQUJvMjR3YkRBT0JnTlZIUThCQWY4RUJBTUNBcVF3RHdZRFZSMFRBUUgvQkFVd0F3RUIKL3pCSkJnTlZIUTRFUWdSQVYyYVU1Nzc0YzhJUHJCMlRJRWRpeUdsdU1xeU5icStKc2dkeWdqRkI5VGJBUzg2OQo2VUJ1TTdmWDdqVS85SzF5TWc4TGJTbTVkSERmUGZXYVdVZ3oxREFOQmdrcWhraUc5dzBCQVFzRkFBT0NBUUVBCkJwMlJpQzdIYnRnUFFJR2Q1RWlsREpYdWFtZ2h6UWJCV0FmREhFeFA0aGtYbXNrNkZMaWxNTDVtT2E0VzNRYlUKclB6RUtyOUdUVHVaa1dtVzRxYnFwUWJKUEtpYllSdUdtMWZBU2phUmtnWjFuN0YwWUFKSGlLK3ZxN2FYcWlBQQpXbHIzOXMxVjZPQ2FYMmhxeXNGcVFJc0xFMnUzcitubjIwS2Y1TWFQQjJqMjNXRjZsS3hiK1pLYTI3SXg3N21lCmoxRUo5VTNCc3hNWmxyZXFUNldxeDdxcEZKYk5sSTBramZubisrZkdDNkltZ3FSYWhOdCtCMTBsMUwwYjMwQWgKRzhMdDl1OHMzSHV2MFdhZzlxUVdyUXdkRGdVN3hFYkNhcmJid3M3aHZTQUxDUFMvbTlZQWpiZEsxeFMzN1VuTAo1NDZxVE1LdmZZNVVaazBIK1BSOWhnPT0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQotLS0tLUJFR0lOIENFUlRJRklDQVRFLS0tLS0KTUlJRm5EQ0NCSVNnQXdJQkFnSVNCa1pvQnlySTBMc05aSHFBR1hpbXpoZ3ZNQTBHQ1NxR1NJYjNEUUVCQ3dVQQpNRE14Q3pBSkJnTlZCQVlUQWxWVE1SWXdGQVlEVlFRS0V3MU1aWFFuY3lCRmJtTnllWEIwTVF3d0NnWURWUVFECkV3TlNNVEF3SGhjTk1qVXdOakl6TVRVd05ERXlXaGNOTWpVd09USXhNVFV3TkRFeFdqQkFNVDR3UEFZRFZRUUQKRERVcUxtRndjSE11Y205ellTNWhkWFJ2TFdSbGRtTnNkWE4wWlhJdVlucGtlQzV3TXk1dmNHVnVjMmhwWm5SaApjSEJ6TG1OdmJUQ0NBU0l3RFFZSktvWklodmNOQVFFQkJRQURnZ0VQQURDQ0FRb0NnZ0VCQUtPM0Q4S3BQL25OClRJKzBldFcrTVM2Q0RyV20yV3dRTnk0VmtNTkNxdkE5N2Z5VVBwMFJIdkh1WkIvbnpxYW9ZN0llN0syVk1xamgKSTZSWVZQc0JBdXROS280SzlLeU9NOG00RHhuTUIyenlTT04wVmRjVHNvMWF4VWUwRU9sb0NqRkxUcG1ya3RwZQpXRDA3T0hCNFV6TnQ1VEN2ajNuMS9mY2xOcUhBN2dYYUdZMk8vWVdtVk0wVHRZM0tWR1BXcmpKQ3pyVlRjSzY0CncxZEhGakNpNC9hbnVocUZDMlhuSU9FME1yaDBxeUJKdHRWa2hyRHlKMkQvNDA2NUpZd1pURTYveWFDQXZQYkwKLzczWm4xeW5PbGJvSDNTenBENWZNL1hpUjVQUHV4T3NON0RkK2NtNDNKR3JEZWo4QURPazlLQmI1dzV2b1lkNgpaaGZvK3M1MXN1TUNBd0VBQWFPQ0Fwc3dnZ0tYTUE0R0ExVWREd0VCL3dRRUF3SUZvREFkQmdOVkhTVUVGakFVCkJnZ3JCZ0VGQlFjREFRWUlLd1lCQlFVSEF3SXdEQVlEVlIwVEFRSC9CQUl3QURBZEJnTlZIUTRFRmdRVW9udUQKdnRobEhmN04ySEh3VUlxM2dqMElFTDh3SHdZRFZSMGpCQmd3Rm9BVXU3ekRSNlhrdktuR3c2UnlEQkNOb2pYaAp5T2d3TXdZSUt3WUJCUVVIQVFFRUp6QWxNQ01HQ0NzR0FRVUZCekFDaGhkb2RIUndPaTh2Y2pFd0xta3ViR1Z1ClkzSXViM0puTHpDQmxRWURWUjBSQklHTk1JR0tnbEVxTGpKcWFtbHNkV0pzWVhRMGMzTnJhMlUyY1Rnek1YVnIKYzJOMk1HMTFkak0yTG5KdmMyRXVZWFYwYnkxa1pYWmpiSFZ6ZEdWeUxtSjZaSGd1Y0RNdWIzQmxibk5vYVdaMApZWEJ3Y3k1amIyMkNOU291WVhCd2N5NXliM05oTG1GMWRHOHRaR1YyWTJ4MWMzUmxjaTVpZW1SNExuQXpMbTl3ClpXNXphR2xtZEdGd2NITXVZMjl0TUJNR0ExVWRJQVFNTUFvd0NBWUdaNEVNQVFJQk1DNEdBMVVkSHdRbk1DVXcKSTZBaG9CK0dIV2gwZEhBNkx5OXlNVEF1WXk1c1pXNWpjaTV2Y21jdk5EZ3VZM0pzTUlJQkJBWUtLd1lCQkFIVwplUUlFQWdTQjlRU0I4Z0R3QUhjQTdUeEwxdWdHd3FTaUFGZmJ5eVRpT0FIZlVTL3R4SWJGY0E4ZzNiYytQK0FBCkFBR1huWWNNWVFBQUJBTUFTREJHQWlFQWxQY3dBNUw4Y1lRSGVXYWs4Nnpva2pRQWNhdVdnSWt4NVNFWjN2ODUKOFZ3Q0lRRDBPMGlXMUZ4ZGlwTi8yUWRsb0RuL09OU0l5aEVXaHFTcEF0SmpKZ0lmTndCMUFOM2N5alNWMStFVwpCZWVWTXZySG4vZzlIRkRmMndBNkZCSjJDaXlzdThncUFBQUJsNTJIREo0QUFBUURBRVl3UkFJZ0dFazE5dGFKCnRPVVNyMlF5eG9rMHNPL3h5NXpkUko3dGJ1OUVkRURKdXhzQ0lGVGlQU1R1b1R1T0ZsdXBGTXdWMHVraG1PRVUKdkxkYTFJbVFWMGpCNlR2Rk1BMEdDU3FHU0liM0RRRUJDd1VBQTRJQkFRQzM1dCtsUjdVZDYvYmU1dmUvbzd0cApTZ3ZOT01SQk12UkZLMnY2OHNpLzJVcDd1N1B3SGdBNmhORHBKWFJpU1A5L0dwR0FFa3FDU2lSd291TnZ3c0d3ClZpbE5kSndQb3p5QVNzUTh2REtBcGorNktBeVYwQmk1VDRvY2VFTDU2b0IvTzFYL2Z0YmZTVTBpTjJRQXRtYmkKUlVPbmhiZE5ML0Evc3cxazgyR1FXaDZSUVVVRVgxVE1FMzErVkZZNW14N1BWUjRkdnVOblhZSkk5bTJhSkYzbgpoN3I5Qy9UaE1XQ243NTNkc3dvQzhmSk9GQnNYcmNVYytFZmQvVEZZbEI2UTA1ZmdlYy9NNDRtemxUQmtMTTc2Cit3U3dMWVFWM2JkTHJMc3hTN1k5TTJkbFhYZzM1aWJwUVBQTGtaUU1odWViWVJhM0UxRElaUGdZaXQ0ZmtvRHUKLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQotLS0tLUJFR0lOIENFUlRJRklDQVRFLS0tLS0KTUlJRkJUQ0NBdTJnQXdJQkFnSVFTNmhTay9lYUw2SnpCa3VvQkkxMTBEQU5CZ2txaGtpRzl3MEJBUXNGQURCUApNUXN3Q1FZRFZRUUdFd0pWVXpFcE1DY0dBMVVFQ2hNZ1NXNTBaWEp1WlhRZ1UyVmpkWEpwZEhrZ1VtVnpaV0Z5ClkyZ2dSM0p2ZFhBeEZUQVRCZ05WQkFNVERFbFRVa2NnVW05dmRDQllNVEFlRncweU5EQXpNVE13TURBd01EQmEKRncweU56QXpNVEl5TXpVNU5UbGFNRE14Q3pBSkJnTlZCQVlUQWxWVE1SWXdGQVlEVlFRS0V3MU1aWFFuY3lCRgpibU55ZVhCME1Rd3dDZ1lEVlFRREV3TlNNVEF3Z2dFaU1BMEdDU3FHU0liM0RRRUJBUVVBQTRJQkR3QXdnZ0VLCkFvSUJBUURQVitYbXhGUVM3YlJIL3NrbldIWkdVQ2lNSFQ2STN3V2QxYlVZS2IzZHRWcS8rdmJPbzc2dkFDRkwKWWxwYVBBRXZ4VmdEOW9uL2poRkQ2OEcxNEJRSGxvOXZIOWZudW9FNUNYVmx0OEt2R0ZzM0ppam5vL1FISzIwYQovNnRZdkpXdVFQL3B5MWZFdFZ0L2VBMFlZYndYNTFUR3UwbVJ6VzRZMFlDRjdxWmxOcngwNnJ4UVRPcjhJZk00CkZwT1V1ckRUYXpnR3pSWVNlc3BTZGNpdGRyTENuRjJZUlZ4dllYdkdMZTQ4RTFLR0FkbFg1amdjMzQyMUg1S1IKbXVkS0hNeEZxSEpWOExEbW93ZnMvYWNiWnA0L1NJdHhoSEZZeVRyNjcxN3lXMFFyUEhUbmo3Skh3UWRxelpxMwpEWmIzRW9FbVVWUUs3R0gyOS9YaThvcklsUTJOQWdNQkFBR2pnZmd3Z2ZVd0RnWURWUjBQQVFIL0JBUURBZ0dHCk1CMEdBMVVkSlFRV01CUUdDQ3NHQVFVRkJ3TUNCZ2dyQmdFRkJRY0RBVEFTQmdOVkhSTUJBZjhFQ0RBR0FRSC8KQWdFQU1CMEdBMVVkRGdRV0JCUzd2TU5IcGVTOHFjYkRwSElNRUkyaU5lSEk2REFmQmdOVkhTTUVHREFXZ0JSNQp0Rm5tZTdibDVBRnpnQWlJeUJwWTl1bWJiakF5QmdnckJnRUZCUWNCQVFRbU1DUXdJZ1lJS3dZQkJRVUhNQUtHCkZtaDBkSEE2THk5NE1TNXBMbXhsYm1OeUxtOXlaeTh3RXdZRFZSMGdCQXd3Q2pBSUJnWm5nUXdCQWdFd0p3WUQKVlIwZkJDQXdIakFjb0JxZ0dJWVdhSFIwY0RvdkwzZ3hMbU11YkdWdVkzSXViM0puTHpBTkJna3Foa2lHOXcwQgpBUXNGQUFPQ0FnRUFrckhuUVRmcmVaMkI1czNpSmVFNklPbVFSSldqZ1Z6UHcxMzl2YUJ3MWJHV0tDSUwwdklvCnp3em4xT1pEakNRaUhjRkNrdEVKcjU5TDlNaHdUeUFXc1ZyZEFmWWYrQjloYXhRbnNIS05ZNjd1NHM1THp6ZmQKdTZQVXplZXRVSzI5ditQc1BtSTJjSmt4cCtpTjNlcGk0aEt1OVp6VVBTd01xdENjZWI3cVBWeEVicFl4WTFwOQoxbjVQSktCTEJYOWViOUxVNmw4elN4UFdWN2JLM2xHNFhhTUpnblQ5eDNpZXM3bXNGdHBLSzViRHRvdGlqL2wwCkdhS2VBOTdwYjV1d0Q5S2dXdmFGWE1JRXQ4alZUakxFdndSZHZDbjI5NEdQREYwOFU4bEFrSXY3dGdobHVhUWgKMVFubEU0U0VONExPRUNqOGRzSUdKWHBHVWszYVUzS2tKejlpY0t5K2FVZ0ErMmNQMjF1aDZOY0RJUzNYeWZhWgpRam1EUTk5M0NoSUk4U1hXdXBRWlZCaUlwY1dPNFJxWmszbHI3Qno1TVVDd3pESUEzNTllNTdTU3E1Q0NrWTBOCjRCNlZ1bGs3TGt0ZndyZEdOVkk1QnNDOXFxeFN3U0tnUkplWjl3eWdJYWVoYkhGSEZoY0JhTURLcGlabEJIeXoKcnNubmxGWENiNXM4SEtuNUxzVWdHdkIyNEw3c0dOWlAyQ1g3ZGhIb3YrWWhEK2pvekxXMnA5VzQ5NTlCejJFaQpSbXFEdG1pWExuenFUcFhiSStzdXlDc29oS1JnNlVuMFJDNDcrY3BpVndIaVhaQVcrY244ZWlOSWpxYlZnWEx4CktQcGR6dnZ0VG5PUGxDN1NRWlNZbWR1bnIzQmY5Yjc3QWlDL1ppZHN0SzM2ZFJJTEt6N09BNTQ9Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
contexts:
- name: test-context
  context:
    cluster: test-cluster
    user: test-user
- name: another-context
  context:
    cluster: test-cluster
    user: test-user
current-context: test-context
users:
- name: test-user
  user:
    token: a2V5LWRhdGE=
`

	invalidKubeconfig = `
apiVersion: v1
kind: Config
invalid: yaml
`

	kubeconfigWithMultipleContexts = `
apiVersion: v1
kind: Config
clusters:
- name: prod-cluster
  cluster:
    server: https://prod-cluster.example.com
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURQRENDQWlTZ0F3SUJBZ0lJVFJIb25HQzdxRGd3RFFZSktvWklodmNOQVFFTEJRQXdKakVTTUJBR0ExVUUKQ3hNSmIzQmxibk5vYVdaME1SQXdEZ1lEVlFRREV3ZHliMjkwTFdOaE1CNFhEVEkxTURZeU16RTJNREkxTWxvWApEVE0xTURZeU1URTJNREkxTWxvd0pqRVNNQkFHQTFVRUN4TUpiM0JsYm5Ob2FXWjBNUkF3RGdZRFZRUURFd2R5CmIyOTBMV05oTUlJQklqQU5CZ2txaGtpRzl3MEJBUUVGQUFPQ0FROEFNSUlCQ2dLQ0FRRUF2WkxDK0FCbFpzWVcKcEpHNUFRYW9PbXlKSDRuSUl3dENiU0dyN1hkQjN6bFZ4aWJITURRWWQ3ZEFiTXc1d1EvalhETlV6OER4RUZKNQpxUjR0SVZRelVSb05tOU9NSU10Rkg2SGhKeFpCVGhSZitmYXY3amh4QVZ3elBlbTV0L09ad2VURHdtdkdoOVNYCkkxUTVUUFVlOHZIaDNPU095TTJTV3Qvckc4UmZhazlucUwydEc5cWhkSkN6Q0xMRGpxdDlZeXJmbUIzbHo0ZUUKcTFJMXViZGVWVlpadjJHWXN6Wll6NTVBOGYxell6aE4rZHVEQ0s2WlhrZVozOFNuY2hHRXFqd21MN3VBK1RkaQpScW9ESlZkRTRmV0lvUTEzUXNBcytWMXF2Q21rWHpOWWxKWFlqMHBXU1dyU3hpaFdTd0lWaG0zRXhuRnhYYnFqCkNFdm1iT3dIeFFJREFRQUJvMjR3YkRBT0JnTlZIUThCQWY4RUJBTUNBcVF3RHdZRFZSMFRBUUgvQkFVd0F3RUIKL3pCSkJnTlZIUTRFUWdSQVYyYVU1Nzc0YzhJUHJCMlRJRWRpeUdsdU1xeU5icStKc2dkeWdqRkI5VGJBUzg2OQo2VUJ1TTdmWDdqVS85SzF5TWc4TGJTbTVkSERmUGZXYVdVZ3oxREFOQmdrcWhraUc5dzBCQVFzRkFBT0NBUUVBCkJwMlJpQzdIYnRnUFFJR2Q1RWlsREpYdWFtZ2h6UWJCV0FmREhFeFA0aGtYbXNrNkZMaWxNTDVtT2E0VzNRYlUKclB6RUtyOUdUVHVaa1dtVzRxYnFwUWJKUEtpYllSdUdtMWZBU2phUmtnWjFuN0YwWUFKSGlLK3ZxN2FYcWlBQQpXbHIzOXMxVjZPQ2FYMmhxeXNGcVFJc0xFMnUzcitubjIwS2Y1TWFQQjJqMjNXRjZsS3hiK1pLYTI3SXg3N21lCmoxRUo5VTNCc3hNWmxyZXFUNldxeDdxcEZKYk5sSTBramZubisrZkdDNkltZ3FSYWhOdCtCMTBsMUwwYjMwQWgKRzhMdDl1OHMzSHV2MFdhZzlxUVdyUXdkRGdVN3hFYkNhcmJid3M3aHZTQUxDUFMvbTlZQWpiZEsxeFMzN1VuTAo1NDZxVE1LdmZZNVVaazBIK1BSOWhnPT0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQotLS0tLUJFR0lOIENFUlRJRklDQVRFLS0tLS0KTUlJRm5EQ0NCSVNnQXdJQkFnSVNCa1pvQnlySTBMc05aSHFBR1hpbXpoZ3ZNQTBHQ1NxR1NJYjNEUUVCQ3dVQQpNRE14Q3pBSkJnTlZCQVlUQWxWVE1SWXdGQVlEVlFRS0V3MU1aWFFuY3lCRmJtTnllWEIwTVF3d0NnWURWUVFECkV3TlNNVEF3SGhjTk1qVXdOakl6TVRVd05ERXlXaGNOTWpVd09USXhNVFV3TkRFeFdqQkFNVDR3UEFZRFZRUUQKRERVcUxtRndjSE11Y205ellTNWhkWFJ2TFdSbGRtTnNkWE4wWlhJdVlucGtlQzV3TXk1dmNHVnVjMmhwWm5SaApjSEJ6TG1OdmJUQ0NBU0l3RFFZSktvWklodmNOQVFFQkJRQURnZ0VQQURDQ0FRb0NnZ0VCQUtPM0Q4S3BQL25OClRJKzBldFcrTVM2Q0RyV20yV3dRTnk0VmtNTkNxdkE5N2Z5VVBwMFJIdkh1WkIvbnpxYW9ZN0llN0syVk1xamgKSTZSWVZQc0JBdXROS280SzlLeU9NOG00RHhuTUIyenlTT04wVmRjVHNvMWF4VWUwRU9sb0NqRkxUcG1ya3RwZQpXRDA3T0hCNFV6TnQ1VEN2ajNuMS9mY2xOcUhBN2dYYUdZMk8vWVdtVk0wVHRZM0tWR1BXcmpKQ3pyVlRjSzY0CncxZEhGakNpNC9hbnVocUZDMlhuSU9FME1yaDBxeUJKdHRWa2hyRHlKMkQvNDA2NUpZd1pURTYveWFDQXZQYkwKLzczWm4xeW5PbGJvSDNTenBENWZNL1hpUjVQUHV4T3NON0RkK2NtNDNKR3JEZWo4QURPazlLQmI1dzV2b1lkNgpaaGZvK3M1MXN1TUNBd0VBQWFPQ0Fwc3dnZ0tYTUE0R0ExVWREd0VCL3dRRUF3SUZvREFkQmdOVkhTVUVGakFVCkJnZ3JCZ0VGQlFjREFRWUlLd1lCQlFVSEF3SXdEQVlEVlIwVEFRSC9CQUl3QURBZEJnTlZIUTRFRmdRVW9udUQKdnRobEhmN04ySEh3VUlxM2dqMElFTDh3SHdZRFZSMGpCQmd3Rm9BVXU3ekRSNlhrdktuR3c2UnlEQkNOb2pYaAp5T2d3TXdZSUt3WUJCUVVIQVFFRUp6QWxNQ01HQ0NzR0FRVUZCekFDaGhkb2RIUndPaTh2Y2pFd0xta3ViR1Z1ClkzSXViM0puTHpDQmxRWURWUjBSQklHTk1JR0tnbEVxTGpKcWFtbHNkV0pzWVhRMGMzTnJhMlUyY1Rnek1YVnIKYzJOMk1HMTFkak0yTG5KdmMyRXVZWFYwYnkxa1pYWmpiSFZ6ZEdWeUxtSjZaSGd1Y0RNdWIzQmxibk5vYVdaMApZWEJ3Y3k1amIyMkNOU291WVhCd2N5NXliM05oTG1GMWRHOHRaR1YyWTJ4MWMzUmxjaTVpZW1SNExuQXpMbTl3ClpXNXphR2xtZEdGd2NITXVZMjl0TUJNR0ExVWRJQVFNTUFvd0NBWUdaNEVNQVFJQk1DNEdBMVVkSHdRbk1DVXcKSTZBaG9CK0dIV2gwZEhBNkx5OXlNVEF1WXk1c1pXNWpjaTV2Y21jdk5EZ3VZM0pzTUlJQkJBWUtLd1lCQkFIVwplUUlFQWdTQjlRU0I4Z0R3QUhjQTdUeEwxdWdHd3FTaUFGZmJ5eVRpT0FIZlVTL3R4SWJGY0E4ZzNiYytQK0FBCkFBR1huWWNNWVFBQUJBTUFTREJHQWlFQWxQY3dBNUw4Y1lRSGVXYWs4Nnpva2pRQWNhdVdnSWt4NVNFWjN2ODUKOFZ3Q0lRRDBPMGlXMUZ4ZGlwTi8yUWRsb0RuL09OU0l5aEVXaHFTcEF0SmpKZ0lmTndCMUFOM2N5alNWMStFVwpCZWVWTXZySG4vZzlIRkRmMndBNkZCSjJDaXlzdThncUFBQUJsNTJIREo0QUFBUURBRVl3UkFJZ0dFazE5dGFKCnRPVVNyMlF5eG9rMHNPL3h5NXpkUko3dGJ1OUVkRURKdXhzQ0lGVGlQU1R1b1R1T0ZsdXBGTXdWMHVraG1PRVUKdkxkYTFJbVFWMGpCNlR2Rk1BMEdDU3FHU0liM0RRRUJDd1VBQTRJQkFRQzM1dCtsUjdVZDYvYmU1dmUvbzd0cApTZ3ZOT01SQk12UkZLMnY2OHNpLzJVcDd1N1B3SGdBNmhORHBKWFJpU1A5L0dwR0FFa3FDU2lSd291TnZ3c0d3ClZpbE5kSndQb3p5QVNzUTh2REtBcGorNktBeVYwQmk1VDRvY2VFTDU2b0IvTzFYL2Z0YmZTVTBpTjJRQXRtYmkKUlVPbmhiZE5ML0Evc3cxazgyR1FXaDZSUVVVRVgxVE1FMzErVkZZNW14N1BWUjRkdnVOblhZSkk5bTJhSkYzbgpoN3I5Qy9UaE1XQ243NTNkc3dvQzhmSk9GQnNYcmNVYytFZmQvVEZZbEI2UTA1ZmdlYy9NNDRtemxUQmtMTTc2Cit3U3dMWVFWM2JkTHJMc3hTN1k5TTJkbFhYZzM1aWJwUVBQTGtaUU1odWViWVJhM0UxRElaUGdZaXQ0ZmtvRHUKLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQotLS0tLUJFR0lOIENFUlRJRklDQVRFLS0tLS0KTUlJRkJUQ0NBdTJnQXdJQkFnSVFTNmhTay9lYUw2SnpCa3VvQkkxMTBEQU5CZ2txaGtpRzl3MEJBUXNGQURCUApNUXN3Q1FZRFZRUUdFd0pWVXpFcE1DY0dBMVVFQ2hNZ1NXNTBaWEp1WlhRZ1UyVmpkWEpwZEhrZ1VtVnpaV0Z5ClkyZ2dSM0p2ZFhBeEZUQVRCZ05WQkFNVERFbFRVa2NnVW05dmRDQllNVEFlRncweU5EQXpNVE13TURBd01EQmEKRncweU56QXpNVEl5TXpVNU5UbGFNRE14Q3pBSkJnTlZCQVlUQWxWVE1SWXdGQVlEVlFRS0V3MU1aWFFuY3lCRgpibU55ZVhCME1Rd3dDZ1lEVlFRREV3TlNNVEF3Z2dFaU1BMEdDU3FHU0liM0RRRUJBUVVBQTRJQkR3QXdnZ0VLCkFvSUJBUURQVitYbXhGUVM3YlJIL3NrbldIWkdVQ2lNSFQ2STN3V2QxYlVZS2IzZHRWcS8rdmJPbzc2dkFDRkwKWWxwYVBBRXZ4VmdEOW9uL2poRkQ2OEcxNEJRSGxvOXZIOWZudW9FNUNYVmx0OEt2R0ZzM0ppam5vL1FISzIwYQovNnRZdkpXdVFQL3B5MWZFdFZ0L2VBMFlZYndYNTFUR3UwbVJ6VzRZMFlDRjdxWmxOcngwNnJ4UVRPcjhJZk00CkZwT1V1ckRUYXpnR3pSWVNlc3BTZGNpdGRyTENuRjJZUlZ4dllYdkdMZTQ4RTFLR0FkbFg1amdjMzQyMUg1S1IKbXVkS0hNeEZxSEpWOExEbW93ZnMvYWNiWnA0L1NJdHhoSEZZeVRyNjcxN3lXMFFyUEhUbmo3Skh3UWRxelpxMwpEWmIzRW9FbVVWUUs3R0gyOS9YaThvcklsUTJOQWdNQkFBR2pnZmd3Z2ZVd0RnWURWUjBQQVFIL0JBUURBZ0dHCk1CMEdBMVVkSlFRV01CUUdDQ3NHQVFVRkJ3TUNCZ2dyQmdFRkJRY0RBVEFTQmdOVkhSTUJBZjhFQ0RBR0FRSC8KQWdFQU1CMEdBMVVkRGdRV0JCUzd2TU5IcGVTOHFjYkRwSElNRUkyaU5lSEk2REFmQmdOVkhTTUVHREFXZ0JSNQp0Rm5tZTdibDVBRnpnQWlJeUJwWTl1bWJiakF5QmdnckJnRUZCUWNCQVFRbU1DUXdJZ1lJS3dZQkJRVUhNQUtHCkZtaDBkSEE2THk5NE1TNXBMbXhsYm1OeUxtOXlaeTh3RXdZRFZSMGdCQXd3Q2pBSUJnWm5nUXdCQWdFd0p3WUQKVlIwZkJDQXdIakFjb0JxZ0dJWVdhSFIwY0RvdkwzZ3hMbU11YkdWdVkzSXViM0puTHpBTkJna3Foa2lHOXcwQgpBUXNGQUFPQ0FnRUFrckhuUVRmcmVaMkI1czNpSmVFNklPbVFSSldqZ1Z6UHcxMzl2YUJ3MWJHV0tDSUwwdklvCnp3em4xT1pEakNRaUhjRkNrdEVKcjU5TDlNaHdUeUFXc1ZyZEFmWWYrQjloYXhRbnNIS05ZNjd1NHM1THp6ZmQKdTZQVXplZXRVSzI5ditQc1BtSTJjSmt4cCtpTjNlcGk0aEt1OVp6VVBTd01xdENjZWI3cVBWeEVicFl4WTFwOQoxbjVQSktCTEJYOWViOUxVNmw4elN4UFdWN2JLM2xHNFhhTUpnblQ5eDNpZXM3bXNGdHBLSzViRHRvdGlqL2wwCkdhS2VBOTdwYjV1d0Q5S2dXdmFGWE1JRXQ4alZUakxFdndSZHZDbjI5NEdQREYwOFU4bEFrSXY3dGdobHVhUWgKMVFubEU0U0VONExPRUNqOGRzSUdKWHBHVWszYVUzS2tKejlpY0t5K2FVZ0ErMmNQMjF1aDZOY0RJUzNYeWZhWgpRam1EUTk5M0NoSUk4U1hXdXBRWlZCaUlwY1dPNFJxWmszbHI3Qno1TVVDd3pESUEzNTllNTdTU3E1Q0NrWTBOCjRCNlZ1bGs3TGt0ZndyZEdOVkk1QnNDOXFxeFN3U0tnUkplWjl3eWdJYWVoYkhGSEZoY0JhTURLcGlabEJIeXoKcnNubmxGWENiNXM4SEtuNUxzVWdHdkIyNEw3c0dOWlAyQ1g3ZGhIb3YrWWhEK2pvekxXMnA5VzQ5NTlCejJFaQpSbXFEdG1pWExuenFUcFhiSStzdXlDc29oS1JnNlVuMFJDNDcrY3BpVndIaVhaQVcrY244ZWlOSWpxYlZnWEx4CktQcGR6dnZ0VG5PUGxDN1NRWlNZbWR1bnIzQmY5Yjc3QWlDL1ppZHN0SzM2ZFJJTEt6N09BNTQ9Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
- name: staging-cluster
  cluster:
    server: https://staging-cluster.example.com
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURQRENDQWlTZ0F3SUJBZ0lJVFJIb25HQzdxRGd3RFFZSktvWklodmNOQVFFTEJRQXdKakVTTUJBR0ExVUUKQ3hNSmIzQmxibk5vYVdaME1SQXdEZ1lEVlFRREV3ZHliMjkwTFdOaE1CNFhEVEkxTURZeU16RTJNREkxTWxvWApEVE0xTURZeU1URTJNREkxTWxvd0pqRVNNQkFHQTFVRUN4TUpiM0JsYm5Ob2FXWjBNUkF3RGdZRFZRUURFd2R5CmIyOTBMV05oTUlJQklqQU5CZ2txaGtpRzl3MEJBUUVGQUFPQ0FROEFNSUlCQ2dLQ0FRRUF2WkxDK0FCbFpzWVcKcEpHNUFRYW9PbXlKSDRuSUl3dENiU0dyN1hkQjN6bFZ4aWJITURRWWQ3ZEFiTXc1d1EvalhETlV6OER4RUZKNQpxUjR0SVZRelVSb05tOU9NSU10Rkg2SGhKeFpCVGhSZitmYXY3amh4QVZ3elBlbTV0L09ad2VURHdtdkdoOVNYCkkxUTVUUFVlOHZIaDNPU095TTJTV3Qvckc4UmZhazlucUwydEc5cWhkSkN6Q0xMRGpxdDlZeXJmbUIzbHo0ZUUKcTFJMXViZGVWVlpadjJHWXN6Wll6NTVBOGYxell6aE4rZHVEQ0s2WlhrZVozOFNuY2hHRXFqd21MN3VBK1RkaQpScW9ESlZkRTRmV0lvUTEzUXNBcytWMXF2Q21rWHpOWWxKWFlqMHBXU1dyU3hpaFdTd0lWaG0zRXhuRnhYYnFqCkNFdm1iT3dIeFFJREFRQUJvMjR3YkRBT0JnTlZIUThCQWY4RUJBTUNBcVF3RHdZRFZSMFRBUUgvQkFVd0F3RUIKL3pCSkJnTlZIUTRFUWdSQVYyYVU1Nzc0YzhJUHJCMlRJRWRpeUdsdU1xeU5icStKc2dkeWdqRkI5VGJBUzg2OQo2VUJ1TTdmWDdqVS85SzF5TWc4TGJTbTVkSERmUGZXYVdVZ3oxREFOQmdrcWhraUc5dzBCQVFzRkFBT0NBUUVBCkJwMlJpQzdIYnRnUFFJR2Q1RWlsREpYdWFtZ2h6UWJCV0FmREhFeFA0aGtYbXNrNkZMaWxNTDVtT2E0VzNRYlUKclB6RUtyOUdUVHVaa1dtVzRxYnFwUWJKUEtpYllSdUdtMWZBU2phUmtnWjFuN0YwWUFKSGlLK3ZxN2FYcWlBQQpXbHIzOXMxVjZPQ2FYMmhxeXNGcVFJc0xFMnUzcitubjIwS2Y1TWFQQjJqMjNXRjZsS3hiK1pLYTI3SXg3N21lCmoxRUo5VTNCc3hNWmxyZXFUNldxeDdxcEZKYk5sSTBramZubisrZkdDNkltZ3FSYWhOdCtCMTBsMUwwYjMwQWgKRzhMdDl1OHMzSHV2MFdhZzlxUVdyUXdkRGdVN3hFYkNhcmJid3M3aHZTQUxDUFMvbTlZQWpiZEsxeFMzN1VuTAo1NDZxVE1LdmZZNVVaazBIK1BSOWhnPT0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQotLS0tLUJFR0lOIENFUlRJRklDQVRFLS0tLS0KTUlJRm5EQ0NCSVNnQXdJQkFnSVNCa1pvQnlySTBMc05aSHFBR1hpbXpoZ3ZNQTBHQ1NxR1NJYjNEUUVCQ3dVQQpNRE14Q3pBSkJnTlZCQVlUQWxWVE1SWXdGQVlEVlFRS0V3MU1aWFFuY3lCRmJtTnllWEIwTVF3d0NnWURWUVFECkV3TlNNVEF3SGhjTk1qVXdOakl6TVRVd05ERXlXaGNOTWpVd09USXhNVFV3TkRFeFdqQkFNVDR3UEFZRFZRUUQKRERVcUxtRndjSE11Y205ellTNWhkWFJ2TFdSbGRtTnNkWE4wWlhJdVlucGtlQzV3TXk1dmNHVnVjMmhwWm5SaApjSEJ6TG1OdmJUQ0NBU0l3RFFZSktvWklodmNOQVFFQkJRQURnZ0VQQURDQ0FRb0NnZ0VCQUtPM0Q4S3BQL25OClRJKzBldFcrTVM2Q0RyV20yV3dRTnk0VmtNTkNxdkE5N2Z5VVBwMFJIdkh1WkIvbnpxYW9ZN0llN0syVk1xamgKSTZSWVZQc0JBdXROS280SzlLeU9NOG00RHhuTUIyenlTT04wVmRjVHNvMWF4VWUwRU9sb0NqRkxUcG1ya3RwZQpXRDA3T0hCNFV6TnQ1VEN2ajNuMS9mY2xOcUhBN2dYYUdZMk8vWVdtVk0wVHRZM0tWR1BXcmpKQ3pyVlRjSzY0CncxZEhGakNpNC9hbnVocUZDMlhuSU9FME1yaDBxeUJKdHRWa2hyRHlKMkQvNDA2NUpZd1pURTYveWFDQXZQYkwKLzczWm4xeW5PbGJvSDNTenBENWZNL1hpUjVQUHV4T3NON0RkK2NtNDNKR3JEZWo4QURPazlLQmI1dzV2b1lkNgpaaGZvK3M1MXN1TUNBd0VBQWFPQ0Fwc3dnZ0tYTUE0R0ExVWREd0VCL3dRRUF3SUZvREFkQmdOVkhTVUVGakFVCkJnZ3JCZ0VGQlFjREFRWUlLd1lCQlFVSEF3SXdEQVlEVlIwVEFRSC9CQUl3QURBZEJnTlZIUTRFRmdRVW9udUQKdnRobEhmN04ySEh3VUlxM2dqMElFTDh3SHdZRFZSMGpCQmd3Rm9BVXU3ekRSNlhrdktuR3c2UnlEQkNOb2pYaAp5T2d3TXdZSUt3WUJCUVVIQVFFRUp6QWxNQ01HQ0NzR0FRVUZCekFDaGhkb2RIUndPaTh2Y2pFd0xta3ViR1Z1ClkzSXViM0puTHpDQmxRWURWUjBSQklHTk1JR0tnbEVxTGpKcWFtbHNkV0pzWVhRMGMzTnJhMlUyY1Rnek1YVnIKYzJOMk1HMTFkak0yTG5KdmMyRXVZWFYwYnkxa1pYWmpiSFZ6ZEdWeUxtSjZaSGd1Y0RNdWIzQmxibk5vYVdaMApZWEJ3Y3k1amIyMkNOU291WVhCd2N5NXliM05oTG1GMWRHOHRaR1YyWTJ4MWMzUmxjaTVpZW1SNExuQXpMbTl3ClpXNXphR2xtZEdGd2NITXVZMjl0TUJNR0ExVWRJQVFNTUFvd0NBWUdaNEVNQVFJQk1DNEdBMVVkSHdRbk1DVXcKSTZBaG9CK0dIV2gwZEhBNkx5OXlNVEF1WXk1c1pXNWpjaTV2Y21jdk5EZ3VZM0pzTUlJQkJBWUtLd1lCQkFIVwplUUlFQWdTQjlRU0I4Z0R3QUhjQTdUeEwxdWdHd3FTaUFGZmJ5eVRpT0FIZlVTL3R4SWJGY0E4ZzNiYytQK0FBCkFBR1huWWNNWVFBQUJBTUFTREJHQWlFQWxQY3dBNUw4Y1lRSGVXYWs4Nnpva2pRQWNhdVdnSWt4NVNFWjN2ODUKOFZ3Q0lRRDBPMGlXMUZ4ZGlwTi8yUWRsb0RuL09OU0l5aEVXaHFTcEF0SmpKZ0lmTndCMUFOM2N5alNWMStFVwpCZWVWTXZySG4vZzlIRkRmMndBNkZCSjJDaXlzdThncUFBQUJsNTJIREo0QUFBUURBRVl3UkFJZ0dFazE5dGFKCnRPVVNyMlF5eG9rMHNPL3h5NXpkUko3dGJ1OUVkRURKdXhzQ0lGVGlQU1R1b1R1T0ZsdXBGTXdWMHVraG1PRVUKdkxkYTFJbVFWMGpCNlR2Rk1BMEdDU3FHU0liM0RRRUJDd1VBQTRJQkFRQzM1dCtsUjdVZDYvYmU1dmUvbzd0cApTZ3ZOT01SQk12UkZLMnY2OHNpLzJVcDd1N1B3SGdBNmhORHBKWFJpU1A5L0dwR0FFa3FDU2lSd291TnZ3c0d3ClZpbE5kSndQb3p5QVNzUTh2REtBcGorNktBeVYwQmk1VDRvY2VFTDU2b0IvTzFYL2Z0YmZTVTBpTjJRQXRtYmkKUlVPbmhiZE5ML0Evc3cxazgyR1FXaDZSUVVVRVgxVE1FMzErVkZZNW14N1BWUjRkdnVOblhZSkk5bTJhSkYzbgpoN3I5Qy9UaE1XQ243NTNkc3dvQzhmSk9GQnNYcmNVYytFZmQvVEZZbEI2UTA1ZmdlYy9NNDRtemxUQmtMTTc2Cit3U3dMWVFWM2JkTHJMc3hTN1k5TTJkbFhYZzM1aWJwUVBQTGtaUU1odWViWVJhM0UxRElaUGdZaXQ0ZmtvRHUKLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQotLS0tLUJFR0lOIENFUlRJRklDQVRFLS0tLS0KTUlJRkJUQ0NBdTJnQXdJQkFnSVFTNmhTay9lYUw2SnpCa3VvQkkxMTBEQU5CZ2txaGtpRzl3MEJBUXNGQURCUApNUXN3Q1FZRFZRUUdFd0pWVXpFcE1DY0dBMVVFQ2hNZ1NXNTBaWEp1WlhRZ1UyVmpkWEpwZEhrZ1VtVnpaV0Z5ClkyZ2dSM0p2ZFhBeEZUQVRCZ05WQkFNVERFbFRVa2NnVW05dmRDQllNVEFlRncweU5EQXpNVE13TURBd01EQmEKRncweU56QXpNVEl5TXpVNU5UbGFNRE14Q3pBSkJnTlZCQVlUQWxWVE1SWXdGQVlEVlFRS0V3MU1aWFFuY3lCRgpibU55ZVhCME1Rd3dDZ1lEVlFRREV3TlNNVEF3Z2dFaU1BMEdDU3FHU0liM0RRRUJBUVVBQTRJQkR3QXdnZ0VLCkFvSUJBUURQVitYbXhGUVM3YlJIL3NrbldIWkdVQ2lNSFQ2STN3V2QxYlVZS2IzZHRWcS8rdmJPbzc2dkFDRkwKWWxwYVBBRXZ4VmdEOW9uL2poRkQ2OEcxNEJRSGxvOXZIOWZudW9FNUNYVmx0OEt2R0ZzM0ppam5vL1FISzIwYQovNnRZdkpXdVFQL3B5MWZFdFZ0L2VBMFlZYndYNTFUR3UwbVJ6VzRZMFlDRjdxWmxOcngwNnJ4UVRPcjhJZk00CkZwT1V1ckRUYXpnR3pSWVNlc3BTZGNpdGRyTENuRjJZUlZ4dllYdkdMZTQ4RTFLR0FkbFg1amdjMzQyMUg1S1IKbXVkS0hNeEZxSEpWOExEbW93ZnMvYWNiWnA0L1NJdHhoSEZZeVRyNjcxN3lXMFFyUEhUbmo3Skh3UWRxelpxMwpEWmIzRW9FbVVWUUs3R0gyOS9YaThvcklsUTJOQWdNQkFBR2pnZmd3Z2ZVd0RnWURWUjBQQVFIL0JBUURBZ0dHCk1CMEdBMVVkSlFRV01CUUdDQ3NHQVFVRkJ3TUNCZ2dyQmdFRkJRY0RBVEFTQmdOVkhSTUJBZjhFQ0RBR0FRSC8KQWdFQU1CMEdBMVVkRGdRV0JCUzd2TU5IcGVTOHFjYkRwSElNRUkyaU5lSEk2REFmQmdOVkhTTUVHREFXZ0JSNQp0Rm5tZTdibDVBRnpnQWlJeUJwWTl1bWJiakF5QmdnckJnRUZCUWNCQVFRbU1DUXdJZ1lJS3dZQkJRVUhNQUtHCkZtaDBkSEE2THk5NE1TNXBMbXhsYm1OeUxtOXlaeTh3RXdZRFZSMGdCQXd3Q2pBSUJnWm5nUXdCQWdFd0p3WUQKVlIwZkJDQXdIakFjb0JxZ0dJWVdhSFIwY0RvdkwzZ3hMbU11YkdWdVkzSXViM0puTHpBTkJna3Foa2lHOXcwQgpBUXNGQUFPQ0FnRUFrckhuUVRmcmVaMkI1czNpSmVFNklPbVFSSldqZ1Z6UHcxMzl2YUJ3MWJHV0tDSUwwdklvCnp3em4xT1pEakNRaUhjRkNrdEVKcjU5TDlNaHdUeUFXc1ZyZEFmWWYrQjloYXhRbnNIS05ZNjd1NHM1THp6ZmQKdTZQVXplZXRVSzI5ditQc1BtSTJjSmt4cCtpTjNlcGk0aEt1OVp6VVBTd01xdENjZWI3cVBWeEVicFl4WTFwOQoxbjVQSktCTEJYOWViOUxVNmw4elN4UFdWN2JLM2xHNFhhTUpnblQ5eDNpZXM3bXNGdHBLSzViRHRvdGlqL2wwCkdhS2VBOTdwYjV1d0Q5S2dXdmFGWE1JRXQ4alZUakxFdndSZHZDbjI5NEdQREYwOFU4bEFrSXY3dGdobHVhUWgKMVFubEU0U0VONExPRUNqOGRzSUdKWHBHVWszYVUzS2tKejlpY0t5K2FVZ0ErMmNQMjF1aDZOY0RJUzNYeWZhWgpRam1EUTk5M0NoSUk4U1hXdXBRWlZCaUlwY1dPNFJxWmszbHI3Qno1TVVDd3pESUEzNTllNTdTU3E1Q0NrWTBOCjRCNlZ1bGs3TGt0ZndyZEdOVkk1QnNDOXFxeFN3U0tnUkplWjl3eWdJYWVoYkhGSEZoY0JhTURLcGlabEJIeXoKcnNubmxGWENiNXM4SEtuNUxzVWdHdkIyNEw3c0dOWlAyQ1g3ZGhIb3YrWWhEK2pvekxXMnA5VzQ5NTlCejJFaQpSbXFEdG1pWExuenFUcFhiSStzdXlDc29oS1JnNlVuMFJDNDcrY3BpVndIaVhaQVcrY244ZWlOSWpxYlZnWEx4CktQcGR6dnZ0VG5PUGxDN1NRWlNZbWR1bnIzQmY5Yjc3QWlDL1ppZHN0SzM2ZFJJTEt6N09BNTQ9Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
contexts:
- name: prod-context
  context:
    cluster: prod-cluster
    user: prod-user
- name: staging-context
  context:
    cluster: staging-cluster
    user: staging-user
current-context: prod-context
users:
- name: prod-user
  user:
    token: a2V5LWRhdGE=
- name: staging-user
  user:
    token: a2V5LWRhdGE=
`
)

func TestNewKubeClient(t *testing.T) {
	kc := NewKubeClient()
	assert.NotNil(t, kc)
}

func TestKubeClient_NewClientFromKubeconfigString(t *testing.T) {
	kc := NewKubeClient()

	t.Run("valid kubeconfig string", func(t *testing.T) {
		client, err := kc.NewClientFromKubeconfigString(validKubeconfig)
		require.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("invalid kubeconfig string", func(t *testing.T) {
		client, err := kc.NewClientFromKubeconfigString(invalidKubeconfig)
		assert.Error(t, err)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "failed to parse kubeconfig string")
	})

	t.Run("empty kubeconfig string", func(t *testing.T) {
		client, err := kc.NewClientFromKubeconfigString("")
		assert.Error(t, err)
		assert.Nil(t, client)
	})
}

func TestKubeClient_NewClientFromKubeconfigStringWithContext(t *testing.T) {
	kc := NewKubeClient()

	t.Run("valid kubeconfig with existing context", func(t *testing.T) {
		client, err := kc.NewClientFromKubeconfigStringWithContext(kubeconfigWithMultipleContexts, "staging-context")
		require.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("valid kubeconfig with default context", func(t *testing.T) {
		client, err := kc.NewClientFromKubeconfigStringWithContext(kubeconfigWithMultipleContexts, "")
		require.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("valid kubeconfig with non-existent context", func(t *testing.T) {
		client, err := kc.NewClientFromKubeconfigStringWithContext(kubeconfigWithMultipleContexts, "non-existent-context")
		assert.Error(t, err)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "context non-existent-context does not exist in kubeconfig")
	})

	t.Run("invalid kubeconfig with context", func(t *testing.T) {
		client, err := kc.NewClientFromKubeconfigStringWithContext(invalidKubeconfig, "test-context")
		assert.Error(t, err)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "failed to parse kubeconfig string")
	})
}

func TestKubeClient_NewClientFromFile(t *testing.T) {
	kc := NewKubeClient()

	// Create a temporary kubeconfig file
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "kubeconfig")
	err := os.WriteFile(kubeconfigPath, []byte(validKubeconfig), 0644)
	require.NoError(t, err)

	t.Run("valid kubeconfig file", func(t *testing.T) {
		client, err := kc.NewClientFromFile(kubeconfigPath)
		require.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("non-existent file", func(t *testing.T) {
		client, err := kc.NewClientFromFile("/non/existent/path")
		assert.Error(t, err)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "kubeconfig file does not exist")
	})

	t.Run("invalid kubeconfig file", func(t *testing.T) {
		invalidPath := filepath.Join(tmpDir, "invalid-kubeconfig")
		err := os.WriteFile(invalidPath, []byte(invalidKubeconfig), 0644)
		require.NoError(t, err)

		client, err := kc.NewClientFromFile(invalidPath)
		assert.Error(t, err)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "failed to load kubeconfig")
	})
}

func TestKubeClient_NewClientFromFileWithContext(t *testing.T) {
	kc := NewKubeClient()

	// Create a temporary kubeconfig file with multiple contexts
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "kubeconfig")
	err := os.WriteFile(kubeconfigPath, []byte(kubeconfigWithMultipleContexts), 0644)
	require.NoError(t, err)

	t.Run("valid kubeconfig file with existing context", func(t *testing.T) {
		client, err := kc.NewClientFromFileWithContext(kubeconfigPath, "staging-context")
		require.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("valid kubeconfig file with default context", func(t *testing.T) {
		client, err := kc.NewClientFromFileWithContext(kubeconfigPath, "")
		require.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("valid kubeconfig file with non-existent context", func(t *testing.T) {
		client, err := kc.NewClientFromFileWithContext(kubeconfigPath, "non-existent-context")
		assert.Error(t, err)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "failed to load kubeconfig")
	})
}

func TestKubeClient_NewClientFromEnv(t *testing.T) {
	kc := NewKubeClient()

	// Create a temporary kubeconfig file
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "kubeconfig")
	err := os.WriteFile(kubeconfigPath, []byte(validKubeconfig), 0644)
	require.NoError(t, err)

	t.Run("with KUBECONFIG environment variable", func(t *testing.T) {
		// Set KUBECONFIG environment variable
		originalKubeconfig := os.Getenv("KUBECONFIG")
		defer func() {
			if err := os.Setenv("KUBECONFIG", originalKubeconfig); err != nil {
				t.Errorf("failed to restore KUBECONFIG: %v", err)
			}
		}()

		if err := os.Setenv("KUBECONFIG", kubeconfigPath); err != nil {
			t.Fatalf("failed to set KUBECONFIG: %v", err)
		}

		client, err := kc.NewClientFromEnv()
		require.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("without KUBECONFIG environment variable", func(t *testing.T) {
		// Clear KUBECONFIG environment variable
		originalKubeconfig := os.Getenv("KUBECONFIG")
		defer func() {
			if err := os.Setenv("KUBECONFIG", originalKubeconfig); err != nil {
				t.Errorf("failed to restore KUBECONFIG: %v", err)
			}
		}()

		if err := os.Unsetenv("KUBECONFIG"); err != nil {
			t.Fatalf("failed to unset KUBECONFIG: %v", err)
		}

		// This should try to use ~/.kube/config which may or may not exist
		// We'll just test that it doesn't panic
		_, err := kc.NewClientFromEnv()
		// We don't assert on the result since ~/.kube/config may not exist in test environment
		if err != nil {
			assert.Contains(t, err.Error(), "kubeconfig file does not exist")
		}
	})
}

func TestKubeClient_NewClientFromEnvWithContext(t *testing.T) {
	kc := NewKubeClient()

	// Create a temporary kubeconfig file with multiple contexts
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "kubeconfig")
	err := os.WriteFile(kubeconfigPath, []byte(kubeconfigWithMultipleContexts), 0644)
	require.NoError(t, err)

	t.Run("with KUBECONFIG environment variable and valid context", func(t *testing.T) {
		// Set KUBECONFIG environment variable
		originalKubeconfig := os.Getenv("KUBECONFIG")
		defer func() {
			if err := os.Setenv("KUBECONFIG", originalKubeconfig); err != nil {
				t.Errorf("failed to restore KUBECONFIG: %v", err)
			}
		}()

		if err := os.Setenv("KUBECONFIG", kubeconfigPath); err != nil {
			t.Fatalf("failed to set KUBECONFIG: %v", err)
		}

		client, err := kc.NewClientFromEnvWithContext("staging-context")
		require.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("with KUBECONFIG environment variable and non-existent context", func(t *testing.T) {
		// Set KUBECONFIG environment variable
		originalKubeconfig := os.Getenv("KUBECONFIG")
		defer func() {
			if err := os.Setenv("KUBECONFIG", originalKubeconfig); err != nil {
				t.Errorf("failed to restore KUBECONFIG: %v", err)
			}
		}()

		if err := os.Setenv("KUBECONFIG", kubeconfigPath); err != nil {
			t.Fatalf("failed to set KUBECONFIG: %v", err)
		}

		client, err := kc.NewClientFromEnvWithContext("non-existent-context")
		assert.Error(t, err)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "failed to load kubeconfig")
	})
}

func TestKubeClient_NewClientFromInCluster(t *testing.T) {
	kc := NewKubeClient()

	// This test will likely fail unless running inside a Kubernetes cluster
	// We'll just test that it doesn't panic and returns an appropriate error
	_, err := kc.NewClientFromInCluster()
	if err != nil {
		// Expected when not running in cluster
		assert.Contains(t, err.Error(), "failed to get in-cluster config")
	}
	// We don't assert on the client since it will be nil when not in cluster
}

func TestKubeClient_NewClientFromConfig(t *testing.T) {
	kc := NewKubeClient()

	// Create a client from a valid kubeconfig string first
	config, err := kc.getConfigWithContext([]byte(validKubeconfig), "")
	require.NoError(t, err)

	t.Run("valid rest.Config", func(t *testing.T) {
		client, err := kc.NewClientFromConfig(config)
		require.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("nil rest.Config", func(t *testing.T) {
		client, err := kc.NewClientFromConfig(nil)
		assert.Error(t, err)
		assert.Nil(t, client)
	})
}

func TestKubeClient_getConfigWithContext(t *testing.T) {
	kc := NewKubeClient()

	t.Run("valid kubeconfig with existing context", func(t *testing.T) {
		config, err := kc.getConfigWithContext([]byte(kubeconfigWithMultipleContexts), "staging-context")
		require.NoError(t, err)
		assert.NotNil(t, config)
		assert.Equal(t, "https://staging-cluster.example.com", config.Host)
	})

	t.Run("valid kubeconfig with default context", func(t *testing.T) {
		config, err := kc.getConfigWithContext([]byte(kubeconfigWithMultipleContexts), "")
		require.NoError(t, err)
		assert.NotNil(t, config)
		assert.Equal(t, "https://prod-cluster.example.com", config.Host)
	})

	t.Run("valid kubeconfig with non-existent context", func(t *testing.T) {
		config, err := kc.getConfigWithContext([]byte(kubeconfigWithMultipleContexts), "non-existent-context")
		assert.Error(t, err)
		assert.Nil(t, config)
		assert.Contains(t, err.Error(), "context non-existent-context does not exist in kubeconfig")
	})

	t.Run("invalid kubeconfig", func(t *testing.T) {
		config, err := kc.getConfigWithContext([]byte(invalidKubeconfig), "test-context")
		assert.Error(t, err)
		assert.Nil(t, config)
		assert.Contains(t, err.Error(), "failed to create client config from bytes")
	})
}

func TestKubeClient_buildConfigFromFileWithContext(t *testing.T) {
	kc := NewKubeClient()

	// Create a temporary kubeconfig file with multiple contexts
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "kubeconfig")
	err := os.WriteFile(kubeconfigPath, []byte(kubeconfigWithMultipleContexts), 0644)
	require.NoError(t, err)

	t.Run("valid file with existing context", func(t *testing.T) {
		config, err := kc.buildConfigFromFileWithContext(kubeconfigPath, "staging-context")
		require.NoError(t, err)
		assert.NotNil(t, config)
		assert.Equal(t, "https://staging-cluster.example.com", config.Host)
	})

	t.Run("valid file with default context", func(t *testing.T) {
		config, err := kc.buildConfigFromFileWithContext(kubeconfigPath, "")
		require.NoError(t, err)
		assert.NotNil(t, config)
		assert.Equal(t, "https://prod-cluster.example.com", config.Host)
	})

	t.Run("valid file with non-existent context", func(t *testing.T) {
		config, err := kc.buildConfigFromFileWithContext(kubeconfigPath, "non-existent-context")
		assert.Error(t, err)
		assert.Nil(t, config)
		assert.Contains(t, err.Error(), "failed to get client config")
	})

	t.Run("non-existent file", func(t *testing.T) {
		config, err := kc.buildConfigFromFileWithContext("/non/existent/path", "test-context")
		assert.Error(t, err)
		assert.Nil(t, config)
		assert.Contains(t, err.Error(), "failed to get client config")
	})
}

func TestKubeClient_createClient(t *testing.T) {
	kc := NewKubeClient()

	// Create a valid config
	config, err := kc.getConfigWithContext([]byte(validKubeconfig), "")
	require.NoError(t, err)

	t.Run("valid config", func(t *testing.T) {
		client, err := kc.createClient(config)
		require.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("nil config", func(t *testing.T) {
		client, err := kc.createClient(nil)
		assert.Error(t, err)
		assert.Nil(t, client)
	})
}

func TestNewClient(t *testing.T) {
	t.Run("default client creation", func(t *testing.T) {
		_, err := NewClient()
		// This may fail if not in cluster and no kubeconfig available
		// We'll just test that it doesn't panic
		if err != nil {
			// Expected when not in cluster and no kubeconfig
			assert.True(t,
				strings.Contains(err.Error(), "kubeconfig file does not exist") ||
					strings.Contains(err.Error(), "failed to get in-cluster config"),
				"Unexpected error: %s", err.Error())
		}
	})
}

func TestNewClientWithContext(t *testing.T) {
	t.Run("client with specific context", func(t *testing.T) {
		_, err := NewClientWithContext("test-context")
		// This may fail if not in cluster and no kubeconfig available
		// We'll just test that it doesn't panic
		if err != nil {
			// Expected when not in cluster and no kubeconfig, or context doesn't exist
			assert.True(t,
				strings.Contains(err.Error(), "kubeconfig file does not exist") ||
					strings.Contains(err.Error(), "context") ||
					strings.Contains(err.Error(), "failed to get in-cluster config"),
				"Unexpected error: %s", err.Error())
		}
	})

	t.Run("client with empty context", func(t *testing.T) {
		_, err := NewClientWithContext("")
		// This may fail if not in cluster and no kubeconfig available
		// We'll just test that it doesn't panic
		if err != nil {
			// Expected when not in cluster and no kubeconfig
			assert.True(t,
				strings.Contains(err.Error(), "kubeconfig file does not exist") ||
					strings.Contains(err.Error(), "failed to get in-cluster config"),
				"Unexpected error: %s", err.Error())
		}
	})
}

func TestNewClientWithGoContext(t *testing.T) {
	t.Run("client with Go context", func(t *testing.T) {
		_, err := NewClientWithGoContext(context.TODO())
		// This may fail if not in cluster and no kubeconfig available
		// We'll just test that it doesn't panic
		if err != nil {
			// Expected when not in cluster and no kubeconfig
			assert.True(t,
				strings.Contains(err.Error(), "kubeconfig file does not exist") ||
					strings.Contains(err.Error(), "failed to get in-cluster config"),
				"Unexpected error: %s", err.Error())
		}
	})
}

// Test that our scheme registration works correctly
func TestSchemeRegistration(t *testing.T) {
	gvkExporterConfigTemplate := schema.GroupVersionKind{Group: "meta.jumpstarter.dev", Version: "v1alpha1", Kind: "ExporterConfigTemplate"}
	gvkExporterHost := schema.GroupVersionKind{Group: "meta.jumpstarter.dev", Version: "v1alpha1", Kind: "ExporterHost"}
	gvkExporterInstance := schema.GroupVersionKind{Group: "meta.jumpstarter.dev", Version: "v1alpha1", Kind: "ExporterInstance"}
	gvkJumpstarterInstance := schema.GroupVersionKind{Group: "meta.jumpstarter.dev", Version: "v1alpha1", Kind: "JumpstarterInstance"}
	gvkPhysicalLocation := schema.GroupVersionKind{Group: "meta.jumpstarter.dev", Version: "v1alpha1", Kind: "PhysicalLocation"}

	_, ok := scheme.Scheme.AllKnownTypes()[gvkExporterConfigTemplate]
	assert.True(t, ok, "ExporterConfigTemplate should be registered in scheme")

	_, ok = scheme.Scheme.AllKnownTypes()[gvkExporterHost]
	assert.True(t, ok, "ExporterHost should be registered in scheme")

	_, ok = scheme.Scheme.AllKnownTypes()[gvkExporterInstance]
	assert.True(t, ok, "ExporterInstance should be registered in scheme")

	_, ok = scheme.Scheme.AllKnownTypes()[gvkJumpstarterInstance]
	assert.True(t, ok, "JumpstarterInstance should be registered in scheme")

	_, ok = scheme.Scheme.AllKnownTypes()[gvkPhysicalLocation]
	assert.True(t, ok, "PhysicalLocation should be registered in scheme")
}
