import { DateTime, Interval } from 'luxon';
import PropTypes from 'prop-types'
import React from 'react'

import TimeGrid from 'react-big-calendar/lib/TimeGrid'
import Week from 'react-big-calendar/lib/Week';

function workWeekRange(date, options) {
  return Week.range(date, options).filter(
    (d) => [0].indexOf(d.getDay()) === -1
  )
}

class WorkWeek extends React.Component {
  render() {
    /**
     * This allows us to default min, max, and scrollToTime
     * using our localizer. This is necessary until such time
     * as TimeGrid is converted to a functional component.
     */
    const {
      date,
      localizer,
      min = localizer.startOf(new Date(), 'day'),
      max = localizer.endOf(new Date(), 'day'),
      scrollToTime = localizer.startOf(new Date(), 'day'),
      enableAutoScroll = true,
      ...props
    } = this.props
    const range = workWeekRange(date, this.props);
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

WorkWeek.defaultProps = TimeGrid.defaultProps

WorkWeek.range = workWeekRange

WorkWeek.navigate = Week.navigate

WorkWeek.title = (date, { formats, culture, ...props }) => {
  const start = DateTime.fromJSDate(date).startOf('week');
  const int = Interval.fromDateTimes(start, start.endOf('week').minus({days: 1}));
  return int.toLocaleString(DateTime.DATE_MED_WITH_WEEKDAY, {
    locale: 'it'
  });
}
export default WorkWeek;