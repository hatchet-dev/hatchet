import { Badge } from '@/components/ui/badge';
import { queries } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import React, { useEffect, useState } from 'react';

interface RecentRequestProps {
  webhookId: string;
  onConnected?: () => void;
  filterBeforeNow?: boolean;
}

const StatusCodeToMessage: Record<number, string> = {
  200: 'Server can receive run requests!',
  401: 'Unauthorized',
  403: 'Forbidden, Check if worker path is correct',
  404: 'Not Found, Check if worker path is correct',
  500: 'Internal Server Error, See worker logs for more information',
  502: 'Bad Gateway, Check if domain is correct and the server is running',
};

export const RecentWebhookRequests: React.FC<RecentRequestProps> = ({
  webhookId,
  onConnected,
  filterBeforeNow = false,
}) => {
  const [timeAfter] = useState(filterBeforeNow ? Date.now() : undefined);
  const [showAll, setShowAll] = useState(false);

  const webhookRequestQuery = useQuery({
    ...queries.webhookWorkers.listRequests(webhookId),
    refetchInterval: 1000,
  });

  const filteredRequests = timeAfter
    ? webhookRequestQuery.data?.requests?.filter(
        (request) => new Date(request.created_at).getTime() > timeAfter,
      )
    : webhookRequestQuery.data?.requests;

  useEffect(() => {
    if (!onConnected) {
      return;
    }

    if (!filteredRequests || filteredRequests.length === 0) {
      return;
    }

    if (filteredRequests[0].statusCode === 200) {
      onConnected();
    }
  }, [onConnected, filteredRequests]);

  if (webhookRequestQuery.isLoading) {
    return <div>Loading...</div>;
  }

  if (webhookRequestQuery.isError) {
    return <div>Error: {webhookRequestQuery.error?.message}</div>;
  }

  if (
    !webhookRequestQuery.data ||
    !webhookRequestQuery.data.requests ||
    webhookRequestQuery.data.requests.length === 0
  ) {
    return <div>Attempting to connect...</div>;
  }

  const requestsToShow = showAll
    ? filteredRequests
    : filteredRequests?.slice(0, 5);

  return (
    <>
      <table className="w-full mb-4">
        {requestsToShow?.map((request, i) => (
          <tr key={i}>
            <td className="font-mono text-gray-500">
              <Badge
                className="mr-4"
                variant={request.statusCode == 200 ? 'successful' : 'failed'}
              >
                {request.statusCode}
              </Badge>
              {request.created_at}
            </td>
            <td>{StatusCodeToMessage[request.statusCode]}</td>
          </tr>
        ))}
      </table>
      {!showAll && webhookRequestQuery.data?.requests.length > 5 && (
        <button onClick={() => setShowAll(true)}>Show More</button>
      )}
    </>
  );
};
