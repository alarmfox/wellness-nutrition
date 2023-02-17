/* eslint-disable @typescript-eslint/no-explicit-any */
/* eslint-disable @typescript-eslint/no-unsafe-assignment */
/* eslint-disable @typescript-eslint/no-unsafe-call */
/* eslint-disable @typescript-eslint/no-unsafe-member-access */
/* eslint-disable @typescript-eslint/ban-ts-comment */
/* eslint-disable @typescript-eslint/no-unsafe-return */
import { DateTime, Interval } from 'luxon';
import type { InferProps } from 'prop-types';
import PropTypes from 'prop-types'
import React from 'react'

// @ts-ignore
import TimeGrid from 'react-big-calendar/lib/TimeGrid';
// @ts-ignore
import Week from 'react-big-calendar/lib/Week';

function workWeekRange(date: Date, options: any) {
  return Week.range(date, options).filter(
    (d: Date) => [0].indexOf(d.getDay()) === -1
  )
}

class WorkWeek extends React.Component {
  static propTypes: { date: PropTypes.Validator<Date>; localizer: PropTypes.Requireable<object>; max: PropTypes.Requireable<Date>; min: PropTypes.Requireable<Date>; scrollToTime: PropTypes.Requireable<Date>; };
  static defaultProps: any;
  static range: (date: Date, options: any) => any;
  static navigate: any;
  static title: (date: any, { formats, culture, ...props }: { [x: string]: any; formats: any; culture: any; }) => string;
  render() {
    /**
     * This allows us to default min, max, and scrollToTime
     * using our localizer. This is necessary until such time
     * as TimeGrid is converted to a functional component.
     */
    // @ts-ignore
    const {
      date,
      localizer,
      // @ts-ignore
      max = localizer.endOf(new Date(), 'day'),
      // @ts-ignore
      min = localizer.startOf(new Date(), 'day'),
      // @ts-ignore
      scrollToTime = localizer.startOf(new Date(), 'day'),
      // @ts-ignore
      enableAutoScroll = true,
      ...props
    }: InferProps<typeof WorkWeek.propTypes> = this.props
    const range = workWeekRange(date , this.props);
    return (
      <TimeGrid
        {...props}
        range={range}
        eventOffset={15}
        localizer={localizer}
        min={min}
        max={max}
        scrollToTime={scrollToTime}
        enableAutoScroll={enableAutoScroll}
      />
    )
  }

}

WorkWeek.propTypes = {
  date: PropTypes.instanceOf(Date).isRequired,
  localizer: PropTypes.object,
  max: PropTypes.instanceOf(Date),
  min: PropTypes.instanceOf(Date),
  scrollToTime: PropTypes.instanceOf(Date),
}

WorkWeek.defaultProps = TimeGrid.defaultProps

WorkWeek.range = workWeekRange

WorkWeek.navigate = Week.navigate

WorkWeek.title = (date: Date, { formats, culture, ...props }) => {
  const start = DateTime.fromJSDate(date).startOf('week');
  const int = Interval.fromDateTimes(start, start.endOf('week').minus({days: 1}));
  return int.toLocaleString(DateTime.DATE_MED_WITH_WEEKDAY, {
    locale: 'it'
  });
}
export default WorkWeek;