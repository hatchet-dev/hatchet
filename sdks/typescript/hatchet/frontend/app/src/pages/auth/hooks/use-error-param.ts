import { useToast } from '@/components/hooks/use-toast';
import { useEffect } from 'react';
import { useSearchParams } from 'react-router-dom';

export default function useErrorParam() {
  const [searchParams, setSearchParams] = useSearchParams();
  const { toast } = useToast();

  useEffect(() => {
    if (searchParams.get('error') && searchParams.get('error') !== '') {
      toast({
        title: 'Error',
        description: searchParams.get('error') || '',
        duration: 5000,
      });

      // remove from search params
      const newSearchParams = new URLSearchParams(searchParams);
      newSearchParams.delete('error');
      setSearchParams(newSearchParams);
    }
  }, [toast, searchParams, setSearchParams]);
}
